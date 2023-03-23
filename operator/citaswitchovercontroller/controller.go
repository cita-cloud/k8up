package citaswitchovercontroller

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

// SwitchoverReconciler reconciles a Fallback object
type SwitchoverReconciler struct {
	Kube client.Client
}

func (f *SwitchoverReconciler) NewObject() *citav1.Switchover {
	return &citav1.Switchover{}
}

func (f *SwitchoverReconciler) NewObjectList() *citav1.SwitchoverList {
	return &citav1.SwitchoverList{}
}

func (f *SwitchoverReconciler) Provision(ctx context.Context, obj *citav1.Switchover) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	config := job.NewConfig(f.Kube, obj, obj.Spec.DestNode)
	executor, err := NewSwitchoverExecutor(ctx, config)
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
		log.Info("switchover just started, waiting...")
		return controllerruntime.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if obj.Status.HasFinished() {
		executor.cleanupOldSwitchover(ctx, obj)
		return controllerruntime.Result{}, nil
	}

	lock := locker.GetForRepository(f.Kube, obj.Spec.DestNode)
	didRun, err := lock.TryRunExclusively(ctx, executor.Execute)
	if !didRun && err == nil {
		log.Info("Delaying cita switchover task, another job is running")
	}
	return controllerruntime.Result{RequeueAfter: time.Second * 30}, err
}

func (f *SwitchoverReconciler) Deprovision(_ context.Context, _ *citav1.Switchover) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}
