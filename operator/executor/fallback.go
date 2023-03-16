package executor

import (
	stderrors "errors"
	"fmt"
	"strconv"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/observer"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type BlockHeightFallbackExecutor struct {
	generic
	bhf  *k8upv1.BlockHeightFallback
	node Node
}

// NewBlockHeightFallbackExecutor returns a new BlockHeightFallbackExecutor.
func NewBlockHeightFallbackExecutor(config job.Config) *BlockHeightFallbackExecutor {
	return &BlockHeightFallbackExecutor{
		generic: generic{config},
	}
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (b *BlockHeightFallbackExecutor) GetConcurrencyLimit() int {
	return cfg.Config.GlobalConcurrentRestoreJobsLimit
}

// Exclusive should return true for jobs that can't run while other jobs run.
func (*BlockHeightFallbackExecutor) Exclusive() bool {
	return true
}

// Execute triggers the actual batch.job creation on the cluster.
// It will also register a callback function on the observer so the PreBackupPods can be removed after the backup has finished.
func (b *BlockHeightFallbackExecutor) Execute() error {
	bhfObject, ok := b.Obj.(*k8upv1.BlockHeightFallback)
	if !ok {
		return stderrors.New("object is not a block height fallback")
	}
	b.bhf = bhfObject

	if b.Obj.GetStatus().Started {
		b.RegisterJobSucceededConditionCallback() // ensure that completed jobs can complete backups between operator restarts.
		return nil
	}

	var err error
	// create node object
	b.node, err = CreateNode(bhfObject.Spec.DeployMethod, bhfObject.Namespace, bhfObject.Spec.Node, b.Client, b.Log)
	if err != nil {
		return err
	}

	genericJob, err := job.GenerateGenericJob(b.Obj, b.Config)
	if err != nil {
		return err
	}

	return b.startBlockHeightFallback(genericJob)
}

func (b *BlockHeightFallbackExecutor) startBlockHeightFallback(job *batchv1.Job) error {
	err := b.node.Stop(b.CTX)
	if err != nil {
		return err
	}
	stopped, err := b.node.CheckStopped(b.CTX)
	if err != nil {
		return err
	}
	if !stopped {
		return nil
	}

	b.registerCITANodeCallback()
	b.RegisterJobSucceededConditionCallback()

	volumes := b.prepareVolumes()

	job.Spec.Template.Spec.Volumes = volumes
	job.Spec.Template.Spec.Containers[0].VolumeMounts = b.newVolumeMounts()

	args, err := b.args()
	if err != nil {
		return err
	}
	job.Spec.Template.Spec.Containers[0].Args = args
	job.Spec.Template.Spec.Containers[0].Command = []string{"/usr/local/bin/k8up", "fallback"}

	if err = b.CreateObjectIfNotExisting(job); err == nil {
		b.SetStarted("the job '%v/%v' was created", job.Namespace, job.Name)
	}
	return err
}

func (b *BlockHeightFallbackExecutor) prepareVolumes() []corev1.Volume {
	vols := make([]corev1.Volume, 0)
	vols = append(vols, corev1.Volume{
		Name: "datadir",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf("datadir-%s-0", b.bhf.Spec.Node),
				ReadOnly:  false,
			},
		}})
	vols = append(vols, corev1.Volume{
		Name: "cita-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-config", b.bhf.Spec.Node),
				},
			},
		}})
	return vols
}

func (b *BlockHeightFallbackExecutor) newVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "datadir",
			MountPath: "/data",
		},
		{
			Name:      "cita-config",
			MountPath: "/cita-config",
		},
	}
}

func (b *BlockHeightFallbackExecutor) args() ([]string, error) {
	crypto, consensus, err := b.GetCryptoAndConsensus(b.bhf.Namespace, b.bhf.Spec.Node)
	if err != nil {
		return nil, err
	}
	return []string{"--block-height", strconv.FormatInt(b.bhf.Spec.BlockHeight, 10),
		"--crypto", crypto,
		"--consensus", consensus}, nil
}

func (b *BlockHeightFallbackExecutor) registerCITANodeCallback() {
	name := b.GetJobNamespacedName()
	observer.GetObserver().RegisterCallback(name.String(), func(_ observer.ObservableJob) {
		//b.StopPreBackupDeployments()
		//b.cleanupOldBackups(name)
		b.startCITANode()
	})
}

func (b *BlockHeightFallbackExecutor) startCITANode() {
	err := b.node.Start(b.CTX)
	if err != nil {
		// todo event
		return
	}
	// todo event
}
