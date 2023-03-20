package controllers_cita

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/handler"
	"github.com/k8up-io/k8up/v2/operator/job"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rivtower.com,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rivtower.com,resources=backups/status;backups/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs="*"
// +kubebuilder:rbac:groups=core,resources=pods/exec,verbs="*"
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;create;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;create;delete

// Reconcile is the entrypoint to manage the given resource.
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cita-backup", req.NamespacedName)

	backup := &citav1.Backup{}
	err := r.Get(ctx, req.NamespacedName, backup)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Backup")
		return ctrl.Result{}, err
	}

	if backup.Status.HasFinished() {
		return ctrl.Result{}, nil
	}

	repository := cfg.Config.GetGlobalRepository()
	if backup.Spec.Backend != nil {
		repository = backup.Spec.Backend.String()
	}
	config := job.NewConfig(ctx, r.Client, log, backup, r.Scheme, repository)

	backupHandler := handler.NewHandler(config)
	return ctrl.Result{RequeueAfter: time.Second * 30}, backupHandler.Handle()
}

// SetupWithManager configures the reconciler.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.Log = l
	return ctrl.NewControllerManagedBy(mgr).
		For(&citav1.Backup{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
