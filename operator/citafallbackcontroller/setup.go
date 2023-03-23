package citafallbackcontroller

import (
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/reconciler"
	batchv1 "k8s.io/api/batch/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

//+kubebuilder:rbac:groups=rivtower.com,resources=blockheightfallbacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rivtower.com,resources=blockheightfallbacks/status,verbs=get;update;patch

// SetupWithManager configures the reconciler.
func SetupWithManager(mgr controllerruntime.Manager) error {
	name := "fallback.rivtower.com"
	r := reconciler.NewReconciler[*citav1.BlockHeightFallback, *citav1.BlockHeightFallbackList](mgr.GetClient(), &FallbackReconciler{
		Kube: mgr.GetClient(),
	})
	return controllerruntime.NewControllerManagedBy(mgr).
		Named(name).
		For(&citav1.BlockHeightFallback{}).
		Owns(&batchv1.Job{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
