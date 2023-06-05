package citarestorecontroller

import (
	batchv1 "k8s.io/api/batch/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/reconciler"
)

// +kubebuilder:rbac:groups=rivtower.com,resources=restores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rivtower.com,resources=restores/status;restores/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager configures the reconciler.
func SetupWithManager(mgr controllerruntime.Manager) error {
	name := "restore.rivtower.com"
	r := reconciler.NewReconciler[*citav1.Restore, *citav1.RestoreList](mgr.GetClient(), &RestoreReconciler{
		Kube: mgr.GetClient(),
	})
	return controllerruntime.NewControllerManagedBy(mgr).
		Named(name).
		For(&citav1.Restore{}).
		Owns(&batchv1.Job{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
