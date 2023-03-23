package controllers_cita

//// PruneReconciler reconciles a Prune object
//type PruneReconciler struct {
//	client.Client
//	Log    logr.Logger
//	Scheme *runtime.Scheme
//}
//
//// +kubebuilder:rbac:groups=rivtower.com,resources=prunes,verbs=get;list;watch;create;update;patch;delete
//// +kubebuilder:rbac:groups=rivtower.com,resources=prunes/status;prunes/finalizers,verbs=get;update;patch
//
//// Reconcile is the entrypoint to manage the given resource.
//func (r *PruneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	log := r.Log.WithValues("cita-prune", req.NamespacedName)
//
//	prune := &citav1.Prune{}
//	err := r.Get(ctx, req.NamespacedName, prune)
//	if err != nil {
//		if errors.IsNotFound(err) {
//			return ctrl.Result{}, nil
//		}
//		return ctrl.Result{}, err
//	}
//
//	if prune.Status.HasFinished() || prune.Status.HasStarted() {
//		return ctrl.Result{}, nil
//	}
//
//	//repository := cfg.Config.GetGlobalRepository()
//	//if prune.Spec.Backend != nil {
//	//	repository = prune.Spec.Backend.String()
//	//}
//	config := job.NewConfig(ctx, r.Client, log, prune, r.Scheme, prune.Spec.Node)
//
//	pruneHandler := handler.NewHandler(config)
//	return ctrl.Result{RequeueAfter: time.Second * 30}, pruneHandler.Handle()
//}
//
//// SetupWithManager configures the reconciler.
//func (r *PruneReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
//	r.Client = mgr.GetClient()
//	r.Scheme = mgr.GetScheme()
//	r.Log = l
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&citav1.Prune{}).
//		WithEventFilter(predicate.GenerationChangedPredicate{}).
//		Complete(r)
//}
