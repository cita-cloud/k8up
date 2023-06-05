package citabackupcontroller

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

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	Kube client.Client
}

func (b *BackupReconciler) NewObject() *citav1.Backup {
	return &citav1.Backup{}
}

func (b *BackupReconciler) NewObjectList() *citav1.BackupList {
	return &citav1.BackupList{}
}

func (b *BackupReconciler) Provision(ctx context.Context, obj *citav1.Backup) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	config := job.NewConfig(b.Kube, obj, obj.Spec.Node)
	executor, err := NewBackupExecutor(ctx, config)
	if err != nil {
		return controllerruntime.Result{}, err
	}

	jobKey := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      executor.jobName(),
	}

	if err := job.ReconcileJobStatus(ctx, jobKey, b.Kube, obj); err != nil {
		return controllerruntime.Result{}, err
	}

	if obj.Status.HasStarted() {
		log.Info("cita backup just started, waiting...")
		return controllerruntime.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if obj.Status.HasFinished() {
		executor.cleanupOldBackups(ctx)
		if obj.Spec.Action == citav1.StopAndStart {
			_ = executor.StartChainNode(ctx)
		}
		return controllerruntime.Result{}, nil
	}

	lock := locker.GetForRepository(b.Kube, obj.Spec.Node)
	didRun, err := lock.TryRunExclusively(ctx, executor.Execute)
	if !didRun && err == nil {
		log.Info("Delaying cita backup task, another job is running")
	}

	return controllerruntime.Result{RequeueAfter: time.Second * 15}, err
}

func (b *BackupReconciler) Deprovision(_ context.Context, _ *citav1.Backup) (controllerruntime.Result, error) {
	return controllerruntime.Result{}, nil
}
