package citafallbackcontroller

import (
	"context"
	"errors"
	"fmt"
	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/domain"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/executor"
	"github.com/k8up-io/k8up/v2/operator/job"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

type FallbackExecutor struct {
	executor.Generic
	fallback *citav1.BlockHeightFallback
	node     domain.Node
}

// NewFallbackExecutor will return a new executor for Fallback jobs.
func NewFallbackExecutor(ctx context.Context, config job.Config) (*FallbackExecutor, error) {

	fallback, ok := config.Obj.(*citav1.BlockHeightFallback)
	if !ok {
		return nil, errors.New("object is not a fallback")
	}
	// create node object
	node, err := domain.CreateNode(fallback.Spec.DeployMethod, fallback.Namespace, fallback.Spec.Node, config.Client,
		controllerruntime.LoggerFrom(ctx))
	if err != nil {
		return nil, err
	}

	return &FallbackExecutor{
		Generic:  executor.Generic{Config: config},
		node:     node,
		fallback: fallback}, nil
}

// GetConcurrencyLimit returns the concurrent jobs limit
func (f *FallbackExecutor) GetConcurrencyLimit() int {
	// todo: unused
	return cfg.Config.GlobalConcurrentRestoreJobsLimit
}

// Execute creates the actual batch.job on the k8s api.
func (f *FallbackExecutor) Execute(ctx context.Context) error {
	log := controllerruntime.LoggerFrom(ctx)

	// wait stop chain node
	stopped, err := f.StopChainNode(ctx)
	if err != nil {
		return err
	}
	if !stopped {
		return nil
	}

	fallbackJob, err := f.createFallbackObject(ctx, f.fallback)
	if err != nil {
		log.Error(err, "unable to create or update fallback object")
		f.SetConditionFalseWithMessage(ctx, k8upv1.ConditionReady, k8upv1.ReasonCreationFailed, "unable to create fallback object: %v", err)
		return nil
	}

	f.SetStarted(ctx, "the job '%v/%v' was created", fallbackJob.Namespace, fallbackJob.Name)

	return nil
}

func (f *FallbackExecutor) cleanupOldFallbacks(ctx context.Context, fallback *citav1.BlockHeightFallback) {
	f.CleanupOldResources(ctx, &citav1.BlockHeightFallbackList{}, fallback.Namespace, fallback)
}

func (f *FallbackExecutor) createFallbackObject(ctx context.Context, fallback *citav1.BlockHeightFallback) (*batchv1.Job, error) {
	batchJob := &batchv1.Job{}
	batchJob.Name = f.jobName()
	batchJob.Namespace = fallback.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, f.Client, batchJob, func() error {
		mutateErr := job.MutateBatchJob(batchJob, fallback, f.Config)
		if mutateErr != nil {
			return mutateErr
		}

		volumes := f.prepareVolumes()

		batchJob.Spec.Template.Spec.Volumes = volumes
		batchJob.Spec.Template.Spec.Containers[0].VolumeMounts = f.newVolumeMounts()

		args, err := f.args(ctx)
		if err != nil {
			return err
		}
		batchJob.Spec.Template.Spec.Containers[0].Args = args
		batchJob.Spec.Template.Spec.Containers[0].Command = []string{"/usr/local/bin/k8up", "fallback"}
		return nil
	})
	return batchJob, err
}

func (f *FallbackExecutor) jobName() string {
	return citav1.FallbackType.String() + "-" + f.Obj.GetName()
}

func (f *FallbackExecutor) StartChainNode(ctx context.Context) {
	log := controllerruntime.LoggerFrom(ctx)
	// start chain node
	err := f.node.Start(ctx)
	if err != nil {
		log.Error(err, "start chain node failed", "name", f.Obj.GetName(), "namespace", f.Obj.GetNamespace())
		f.SetConditionFalseWithMessage(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonFailed, "start chain node failed: %v", err)
		return
	}
	f.SetConditionTrue(ctx, citav1.ConditionStartChainNodeReady, k8upv1.ReasonReady)
	return
}

func (f *FallbackExecutor) StopChainNode(ctx context.Context) (bool, error) {
	log := controllerruntime.LoggerFrom(ctx)
	// stop chain node
	err := f.node.Stop(ctx)
	if err != nil {
		log.Error(err, "stop chain node failed", "name", f.Obj.GetName(), "namespace", f.Obj.GetNamespace())
		f.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "stop chain node failed: %v", err)
		return false, err
	}
	stopped, err := f.node.CheckStopped(ctx)
	if err != nil {
		log.Error(err, "check chain node stopped failed", "name", f.Obj.GetName(), "namespace", f.Obj.GetNamespace())
		f.SetConditionFalseWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonFailed, "check chain node stopped failed: %v", err)
		return false, err
	}
	if !stopped {
		f.SetConditionUnknownWithMessage(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonWaiting, "waiting for chain node stopped")
		return false, nil
	}
	f.SetConditionTrue(ctx, citav1.ConditionStopChainNodeReady, k8upv1.ReasonReady)
	return true, nil
}

func (f *FallbackExecutor) prepareVolumes() []corev1.Volume {
	vols := make([]corev1.Volume, 0)
	vols = append(vols, corev1.Volume{
		Name: "datadir",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf("datadir-%s-0", f.fallback.Spec.Node),
				ReadOnly:  false,
			},
		}})
	vols = append(vols, corev1.Volume{
		Name: "cita-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-config", f.fallback.Spec.Node),
				},
			},
		}})
	return vols
}

func (f *FallbackExecutor) newVolumeMounts() []corev1.VolumeMount {
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

func (f *FallbackExecutor) args(ctx context.Context) ([]string, error) {
	crypto, consensus, err := f.node.GetCryptoAndConsensus(ctx)
	if err != nil {
		return nil, err
	}
	return []string{"--block-height", strconv.FormatInt(f.fallback.Spec.BlockHeight, 10),
		"--crypto", crypto,
		"--consensus", consensus}, nil
}
