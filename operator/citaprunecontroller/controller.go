package citaprunecontroller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/locker"
)

// PruneReconciler reconciles a Prune object
type PruneReconciler struct {
	Kube client.Client
}

func (r *PruneReconciler) NewObject() *citav1.Prune {
	return &citav1.Prune{}
}

func (r *PruneReconciler) NewObjectList() *citav1.PruneList {
	return &citav1.PruneList{}
}

func (r *PruneReconciler) Provision(ctx context.Context, obj *citav1.Prune) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	config := job.NewConfig(r.Kube, obj, obj.Spec.Node)
	executor := NewPruneExecutor(config)

	jobKey := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      executor.jobName(),
	}
	if err := job.ReconcileJobStatus(ctx, jobKey, r.Kube, obj); err != nil {
		return controllerruntime.Result{}, err
	}

	if obj.Status.HasStarted() {
		log.V(1).Info("prune just started, waiting...")
		return controllerruntime.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if obj.Status.HasFinished() {
		executor.cleanupOldPrunes(ctx, obj)
		return controllerruntime.Result{}, nil
	}

	lock := locker.GetForRepository(r.Kube, obj.Spec.Node)
	didRun, err := lock.TryRunExclusively(ctx, executor.Execute)
	if !didRun && err == nil {
		log.Info("Delaying prune task, another job is running")
	}
	return controllerruntime.Result{RequeueAfter: time.Second * 8}, err
}

func (r *PruneReconciler) Deprovision(_ context.Context, _ *citav1.Prune) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}
