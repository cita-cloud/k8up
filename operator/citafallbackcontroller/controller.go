package citafallbackcontroller

import (
	"context"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/locker"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// FallbackReconciler reconciles a Fallback object
type FallbackReconciler struct {
	Kube client.Client
}

func (f *FallbackReconciler) NewObject() *citav1.BlockHeightFallback {
	return &citav1.BlockHeightFallback{}
}

func (f *FallbackReconciler) NewObjectList() *citav1.BlockHeightFallbackList {
	return &citav1.BlockHeightFallbackList{}
}

func (f *FallbackReconciler) Provision(ctx context.Context, obj *citav1.BlockHeightFallback) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	config := job.NewConfig(f.Kube, obj, obj.Spec.Node)
	executor, err := NewFallbackExecutor(ctx, config)
	if err != nil {
		return controllerruntime.Result{}, err
	}

	jobKey := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      executor.jobName(),
	}
	if err := job.ReconcileJobStatus(ctx, jobKey, f.Kube, obj); err != nil {
		return controllerruntime.Result{}, err
	}

	if obj.Status.HasStarted() {
		log.Info("fallback just started, waiting...")
		return controllerruntime.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if obj.Status.HasFinished() {
		executor.cleanupOldFallbacks(ctx, obj)
		return controllerruntime.Result{}, nil
	}

	lock := locker.GetForRepository(f.Kube, obj.Spec.Node)
	didRun, err := lock.TryRunExclusively(ctx, executor.Execute)
	if !didRun && err == nil {
		log.Info("Delaying cita fallback task, another job is running")
	}
	return controllerruntime.Result{RequeueAfter: time.Second * 30}, err
}

func (f *FallbackReconciler) Deprovision(_ context.Context, _ *citav1.BlockHeightFallback) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}
