package controllers_cita

//// RestoreReconciler reconciles a Restore object
//type RestoreReconciler struct {
//	client.Client
//	Log    logr.Logger
//	Scheme *runtime.Scheme
//}
//
//// +kubebuilder:rbac:groups=rivtower.com,resources=restores,verbs=get;list;watch;create;update;patch;delete
//// +kubebuilder:rbac:groups=rivtower.com,resources=restores/status;restores/finalizers,verbs=get;update;patch
//
//// Reconcile is the entrypoint to manage the given resource.
//func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	log := r.Log.WithValues("cita-restore", req.NamespacedName)
//
//	restore := &citav1.Restore{}
//	err := r.Get(ctx, req.NamespacedName, restore)
//	if err != nil {
//		if errors.IsNotFound(err) {
//			return ctrl.Result{}, nil
//		}
//		return ctrl.Result{}, err
//	}
//
//	if restore.Status.HasFinished() || restore.Status.HasStarted() {
//		return ctrl.Result{}, nil
//	}
//
//	//repository := cfg.Config.GetGlobalRepository()
//	//if restore.Spec.Backend != nil {
//	//	repository = restore.Spec.Backend.String()
//	//}
//	config := job.NewConfig(ctx, r.Client, log, restore, r.Scheme, restore.Spec.Node)
//
//	restoreHandler := handler.NewHandler(config)
//	return ctrl.Result{RequeueAfter: time.Second * 30}, restoreHandler.Handle()
//}
//
//// SetupWithManager configures the reconciler.
//func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
//	r.Client = mgr.GetClient()
//	r.Scheme = mgr.GetScheme()
//	r.Log = l
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&citav1.Restore{}).
//		WithEventFilter(predicate.GenerationChangedPredicate{}).
//		Complete(r)
//}
