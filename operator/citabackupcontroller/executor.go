package citabackupcontroller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/domain"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/citabase"
	"github.com/k8up-io/k8up/v2/operator/executor"
	"github.com/k8up-io/k8up/v2/operator/job"
)

// BackupExecutor creates a batch.job object on the cluster. It merges all the
// information provided by defaults and the CRDs to ensure the backup has all information to run.
type BackupExecutor struct {
	citabase.CITABase
	backup *citav1.Backup
}

// NewBackupExecutor returns a new BackupExecutor.
func NewBackupExecutor(ctx context.Context, config job.Config) (*BackupExecutor, error) {
	backup := config.Obj.(*citav1.Backup)
	// create node object
	node, err := domain.CreateNode(backup.Spec.DeployMethod, backup.Namespace, backup.Spec.Node, config.Client,
		controllerruntime.LoggerFrom(ctx))
	if err != nil {
		return nil, err
	}

	citaBase := citabase.NewCITABase(config, node)
	return &BackupExecutor{
		CITABase: citaBase,
		backup:   config.Obj.(*citav1.Backup),
	}, nil
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (b *BackupExecutor) GetConcurrencyLimit() int {
	return cfg.Config.GlobalConcurrentBackupJobsLimit
}

// Execute triggers the actual batch.job creation on the cluster.
// It will also register a callback function on the observer so the PreBackupPods can be removed after the backup has finished.
func (b *BackupExecutor) Execute(ctx context.Context) error {
	err := b.createServiceAccountAndBinding(ctx)
	if err != nil {
		return err
	}

	return b.startBackup(ctx)
}

func (b *BackupExecutor) startBackup(ctx context.Context) error {
	if b.backup.Spec.Action == citav1.StopAndStart {
		// wait stop chain node
		stopped, err := b.StopChainNode(ctx)
		if err != nil {
			return err
		}
		if !stopped {
			return nil
		}
	}

	batchJob := b.createJob()
	_, err := controllerruntime.CreateOrUpdate(ctx, b.Generic.Config.Client, batchJob, func() error {
		mutateErr := job.MutateBatchJob(batchJob, b.backup, b.Generic.Config)
		if mutateErr != nil {
			return mutateErr
		}

		volumes, err := b.prepareVolumes(ctx)
		if err != nil {
			b.SetConditionFalseWithMessage(ctx, k8upv1.ConditionReady, k8upv1.ReasonRetrievalFailed, err.Error())
			return err
		}

		vars, setupErr := b.setupEnvVars()
		if setupErr != nil {
			return setupErr
		}
		batchJob.Spec.Template.Spec.Containers[0].Env = vars
		b.backup.Spec.AppendEnvFromToContainer(&batchJob.Spec.Template.Spec.Containers[0])

		batchJob.Spec.Template.Spec.ServiceAccountName = cfg.Config.ServiceAccount
		batchJob.Spec.Template.Spec.Containers[0].Args = executor.BuildTagArgs(b.backup.Spec.Tags)
		batchJob.Spec.Template.Spec.Volumes = volumes

		if b.backup.Spec.DataType.Full != nil {
			batchJob.Spec.Template.Spec.Containers[0].VolumeMounts = b.newVolumeMountsForFull()
		}
		if b.backup.Spec.DataType.State != nil {
			batchJob.Spec.Template.Spec.Containers[0].VolumeMounts = b.newVolumeMountsForState()
		}
		if b.backup.Spec.Backend.Local != nil {
			if b.backup.Spec.Backend.Local.StorageClass != "" {
				// mount new pvc
				batchJob.Spec.Template.Spec.Containers[0].VolumeMounts = append(batchJob.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      "backup-dest",
					ReadOnly:  false,
					MountPath: b.backup.Spec.Backend.Local.MountPath,
				})
			}
			if b.backup.Spec.Backend.Local.Pvc != "" {
				// mount the pvc provided by the user
				mountPath, err := GetMountPoint(b.backup.Spec.Backend.Local.MountPath)
				if err != nil {
					return nil
				}
				batchJob.Spec.Template.Spec.Containers[0].VolumeMounts = append(batchJob.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      "backup-dest",
					ReadOnly:  false,
					MountPath: mountPath,
				})
			}
		}

		args, err := b.args(ctx)
		if err != nil {
			return err
		}
		batchJob.Spec.Template.Spec.Containers[0].Args = args
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to createOrUpdate(%q): %w", batchJob.Name, err)
	}
	return nil
}

func (b *BackupExecutor) createJob() *batchv1.Job {
	batchJob := &batchv1.Job{}
	batchJob.Name = b.jobName()
	batchJob.Namespace = b.backup.Namespace
	batchJob.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	return batchJob
}

func (b *BackupExecutor) jobName() string {
	return citav1.CITABackupType.String() + "-" + b.Obj.GetName()
}

func (b *BackupExecutor) cleanupOldBackups(ctx context.Context) {
	b.Generic.CleanupOldResources(ctx, &citav1.BackupList{}, b.backup.Namespace, b.backup)
}

func (b *BackupExecutor) prepareVolumes(ctx context.Context) ([]corev1.Volume, error) {
	volumes := make([]corev1.Volume, 0)
	sourceVolume, err := b.GetVolume(ctx)
	if err != nil {
		return nil, err
	}
	volumes = append(volumes, sourceVolume)
	if b.backup.Spec.Backend.Local != nil {
		if b.backup.Spec.Backend.Local.StorageClass != "" {
			destPVC, err := b.createLocalPVC(ctx)
			if err != nil {
				return nil, err
			}
			// add to volumes
			volumes = append(volumes, corev1.Volume{
				Name: "backup-dest",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: destPVC.Name,
						ReadOnly:  false,
					},
				}})
		} else if b.backup.Spec.Backend.Local.Pvc != "" {
			volumes = append(volumes, corev1.Volume{
				Name: "backup-dest",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: b.backup.Spec.Backend.Local.Pvc,
						ReadOnly:  false,
					},
				},
			})
		}

	}
	if b.backup.Spec.DataType.State != nil && b.backup.Spec.DeployMethod == citav1.CloudConfig {
		volumes = append(volumes, corev1.Volume{
			Name: "cita-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-config", b.backup.Spec.Node),
					},
				},
			},
		})
	}
	return volumes, nil
}

func (b *BackupExecutor) createLocalPVC(ctx context.Context) (*corev1.PersistentVolumeClaim, error) {
	var err error
	var resourceRequirements corev1.ResourceRequirements
	if b.backup.Spec.Backend.Local.Size != "" {
		// Create pvc of specified size
		resourceRequirements = corev1.ResourceRequirements{
			Limits:   nil,
			Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(b.backup.Spec.Backend.Local.Size)},
		}
	} else {
		// Create a pvc of the same size as the original
		resourceRequirements, err = b.GetPVCInfo(ctx)
		if err != nil {
			return nil, nil
		}
	}

	destPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.backup.Name,
			Namespace: b.backup.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        resourceRequirements,
			StorageClassName: pointer.String(b.backup.Spec.Backend.Local.StorageClass),
		},
	}
	err = controllerruntime.SetControllerReference(b.backup, destPVC, b.Config.Client.Scheme())
	if err != nil {
		return nil, err
	}
	err = b.CreateObjectIfNotExisting(ctx, destPVC)
	if err != nil {
		return nil, err
	}
	return destPVC, nil
}

func (b *BackupExecutor) newVolumeMountsForFull() []corev1.VolumeMount {
	if b.backup.Spec.DeployMethod == citav1.CloudConfig {
		return []corev1.VolumeMount{
			{
				Name:      "backup-source",
				MountPath: "/data/backup-source",
				ReadOnly:  true,
			},
		}
	}
	// python chain node
	return []corev1.VolumeMount{
		{
			Name:      "backup-source",
			MountPath: "/data/backup-source",
			ReadOnly:  true,
			SubPath:   b.backup.Spec.Node,
		},
	}
}

func (b *BackupExecutor) newVolumeMountsForState() []corev1.VolumeMount {
	if b.backup.Spec.DeployMethod == citav1.CloudConfig {
		return []corev1.VolumeMount{
			{
				Name:      "backup-source",
				MountPath: "/data/backup-source",
				ReadOnly:  false,
			},
			{
				Name:      "cita-config",
				MountPath: "/cita-config",
				ReadOnly:  true,
			},
		}
	}
	return []corev1.VolumeMount{
		{
			Name:      "backup-source",
			MountPath: "/data/backup-source",
			SubPath:   b.backup.Spec.Node,
			ReadOnly:  false,
		},
	}
}

func (b *BackupExecutor) args(ctx context.Context) ([]string, error) {
	var args []string
	if len(b.backup.Spec.Tags) > 0 {
		args = append(args, executor.BuildTagArgs(b.backup.Spec.Tags)...)
	}
	crypto, consensus, err := b.GetCryptoAndConsensus(ctx)
	if err != nil {
		return nil, err
	}
	switch {
	case b.backup.Spec.DataType.Full != nil:
		args = append(args, "-dataType", "full")
		args = append(args, executor.BuildIncludePathArgs(b.backup.Spec.DataType.Full.IncludePaths)...)
	case b.backup.Spec.DataType.State != nil:
		args = append(args, "-dataType", "state")
		args = append(args, "-blockHeight", strconv.FormatInt(b.backup.Spec.DataType.State.BlockHeight, 10))
		args = append(args, "-crypto", crypto)
		args = append(args, "-consensus", consensus)
		args = append(args, "-backupDir", "/state_data")
		args = append(args, "-nodeDeployMethod", string(b.backup.Spec.DeployMethod))
	default:
		return nil, fmt.Errorf("undefined backup data type on '%v/%v'", b.backup.Namespace, b.backup.Name)
	}
	return args, nil
}

func GetMountPoint(path string) (string, error) {
	res := strings.Split(path, "/")
	if len(res) < 2 {
		return "", fmt.Errorf("path invaild")
	}
	return fmt.Sprintf("/%s", res[1]), nil
}
