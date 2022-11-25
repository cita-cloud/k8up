package citarestorecontroller

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

// RestoreReconciler reconciles a Restore object
type RestoreReconciler struct {
	Kube client.Client
}

func (r *RestoreReconciler) NewObject() *citav1.Restore {
	return &citav1.Restore{}
}

func (r *RestoreReconciler) NewObjectList() *citav1.RestoreList {
	return &citav1.RestoreList{}
}

func (r *RestoreReconciler) Provision(ctx context.Context, obj *citav1.Restore) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	config := job.NewConfig(r.Kube, obj, obj.Spec.Node)
	executor, err := NewRestoreExecutor(ctx, config)
	if err != nil {
		return controllerruntime.Result{}, err
	}

	jobKey := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      executor.jobName(),
	}
	if err := job.ReconcileJobStatus(ctx, jobKey, r.Kube, obj); err != nil {
		return controllerruntime.Result{}, err
	}

	if obj.Status.HasStarted() {
		log.Info("restore just started, waiting...")
		return controllerruntime.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if obj.Status.HasFinished() {
		executor.cleanupOldRestores(ctx, obj)
		if obj.Spec.Action == citav1.StopAndStart {
			_ = executor.StartChainNode(ctx)
		}
		return controllerruntime.Result{}, nil
	}

	lock := locker.GetForRepository(r.Kube, obj.Spec.Node)
	didRun, err := lock.TryRunExclusively(ctx, executor.Execute)
	if !didRun && err == nil {
		log.Info("Delaying cita restore task, another job is running")
	}
	return controllerruntime.Result{RequeueAfter: time.Second * 15}, err
}

func (r *RestoreReconciler) Deprovision(_ context.Context, _ *citav1.Restore) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}
