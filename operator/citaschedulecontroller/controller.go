package citaschedulecontroller

import (
	"context"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/scheduler"
)

// ScheduleReconciler reconciles a Schedule object
type ScheduleReconciler struct {
	Kube client.Client
}

func (r *ScheduleReconciler) NewObject() *citav1.Schedule {
	return &citav1.Schedule{}
}

func (r *ScheduleReconciler) NewObjectList() *citav1.ScheduleList {
	return &citav1.ScheduleList{}
}

func (r *ScheduleReconciler) Provision(ctx context.Context, schedule *citav1.Schedule) (controllerruntime.Result, error) {
	log := controllerruntime.LoggerFrom(ctx)

	//if schedule.Spec.Archive != nil && schedule.Spec.Archive.RestoreSpec == nil {
	//	schedule.Spec.Archive.RestoreSpec = &k8upv1.RestoreSpec{}
	//}
	config := job.NewConfig(r.Kube, schedule, schedule.Spec.Node)

	return controllerruntime.Result{}, NewScheduleHandler(config, schedule, log).Handle(ctx)
}

func (r *ScheduleReconciler) Deprovision(ctx context.Context, obj *citav1.Schedule) (controllerruntime.Result, error) {
	for _, jobType := range []k8upv1.JobType{citav1.CITAPruneType, citav1.CITARestoreType, citav1.CITABackupType} {
		key := keyOf(obj, jobType)
		scheduler.GetScheduler().RemoveSchedule(ctx, key)
	}
	controllerutil.RemoveFinalizer(obj, k8upv1.ScheduleFinalizerName)
	return controllerruntime.Result{}, r.Kube.Update(ctx, obj)
}
