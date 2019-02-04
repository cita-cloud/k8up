package restore

import (
	"fmt"
	"time"

	backupv1alpha1 "github.com/vshn/k8up/apis/backup/v1alpha1"
	"github.com/vshn/k8up/service"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRestoreJob(restore *backupv1alpha1.Restore, config config) *batchv1.Job {

	volumes := []corev1.Volume{}
	if restore.Spec.RestoreMethod.S3 == nil {
		volumes = append(volumes,
			corev1.Volume{
				Name: restore.Spec.RestoreMethod.Folder.ClaimName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: restore.Spec.RestoreMethod.Folder.PersistentVolumeClaimVolumeSource,
				},
			})
	}

	mounts := make([]corev1.VolumeMount, 0)
	for _, volume := range volumes {
		tmpMount := corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: service.RestorePath,
		}
		mounts = append(mounts, tmpMount)
	}

	args := []string{"-restore"}

	if restore.Spec.RestoreFilter != "" {
		args = append(args, "-restoreFilter", restore.Spec.RestoreFilter)
	}

	if restore.Spec.Snapshot != "" {
		args = append(args, "-restoreSnap", restore.Spec.Snapshot)
	}

	methodDefined := false
	if restore.Spec.RestoreMethod.Folder != nil {
		args = append(args, "-restoreType", "folder")
		methodDefined = true
	}

	if !methodDefined && restore.Spec.RestoreMethod.S3 != nil {
		args = append(args, "-restoreType", "s3")
		methodDefined = true
	}

	jobName := fmt.Sprintf("restorejob-%v", time.Now().Unix())
	podName := fmt.Sprintf("restorepod-%v", time.Now().Unix())

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: restore.Namespace,
			Labels: map[string]string{
				config.Label: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				service.NewOwnerReference(restore, backupv1alpha1.RestoreKind),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: restore.Namespace,
					Labels: map[string]string{
						config.Label: "true",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicy(config.RestartPolicy),
					Volumes:       volumes,
					Containers: []corev1.Container{
						{
							Name:            podName,
							Image:           config.image,
							VolumeMounts:    mounts,
							Env:             setUpEnvVariables(restore, config),
							ImagePullPolicy: corev1.PullAlways,
							TTY:             true,
							Stdin:           true,
							Args:            args,
						},
					},
				},
			},
		},
	}
}

func setUpEnvVariables(restore *backupv1alpha1.Restore, config config) []corev1.EnvVar {
	vars := service.DefaultEnvs(restore.Spec.Backend, config.Global)

	vars = append(vars, restore.Spec.Backend.S3.RestoreEnvs(config.Global)...)

	return vars
}