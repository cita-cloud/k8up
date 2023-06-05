package citaprunecontroller

import (
	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/reconciler"
)

// +kubebuilder:rbac:groups=rivtower.com,resources=prunes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rivtower.com,resources=prunes/status;prunes/finalizers,verbs=get;update;patch

// SetupWithManager configures the reconciler.
func SetupWithManager(mgr ctrl.Manager) error {
	name := "prune.rivtower.com"
	r := reconciler.NewReconciler[*citav1.Prune, *citav1.PruneList](mgr.GetClient(), &PruneReconciler{
		Kube: mgr.GetClient(),
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&citav1.Prune{}).
		Owns(&batchv1.Job{}).
		Named(name).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
