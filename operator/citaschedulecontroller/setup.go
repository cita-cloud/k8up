package citaschedulecontroller

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/reconciler"
)

// +kubebuilder:rbac:groups=rivtower.com,resources=schedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rivtower.com,resources=schedules/status;schedules/finalizers,verbs=get;update;patch
// The following permissions are just for backwards compatibility.
// +kubebuilder:rbac:groups=k8up.io,resources=effectiveschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8up.io,resources=effectiveschedules/finalizers,verbs=update

// SetupWithManager configures the reconciler.
func SetupWithManager(mgr ctrl.Manager) error {
	name := "schedule.rivtower.com"
	r := reconciler.NewReconciler[*citav1.Schedule, *citav1.ScheduleList](mgr.GetClient(), &ScheduleReconciler{
		Kube: mgr.GetClient(),
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&citav1.Schedule{}).
		Named(name).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
