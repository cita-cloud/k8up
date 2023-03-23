package controllers_cita

//// ScheduleReconciler reconciles a Schedule object
//type ScheduleReconciler struct {
//	client.Client
//	Log    logr.Logger
//	Scheme *runtime.Scheme
//}
//
//// +kubebuilder:rbac:groups=rivtower.com,resources=schedules,verbs=get;list;watch;create;update;patch;delete
//// +kubebuilder:rbac:groups=rivtower.com,resources=schedules/status;schedules/finalizers,verbs=get;update;patch
//// +kubebuilder:rbac:groups=k8up.io,resources=effectiveschedules,verbs=get;list;watch;create;update;patch;delete
//// +kubebuilder:rbac:groups=k8up.io,resources=effectiveschedules/finalizers,verbs=update
//
//// Reconcile is the entrypoint to manage the given resource.
//func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	log := r.Log.WithValues("cita-schedule", req.NamespacedName)
//
//	schedule := &citav1.Schedule{}
//	err := r.Client.Get(ctx, req.NamespacedName, schedule)
//	if err != nil {
//		if errors.IsNotFound(err) {
//			return reconcile.Result{}, nil
//		}
//
//		return reconcile.Result{}, err
//	}
//
//	effectiveSchedules, err := r.fetchEffectiveSchedules(ctx, schedule)
//	if err != nil {
//		requeueAfter := 60 * time.Second
//		r.Log.Info("could not retrieve list of effective schedules", "error", err.Error(), "retry_after", requeueAfter)
//		return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, err
//	}
//
//	//repository := cfg.Config.GetGlobalRepository()
//	//if schedule.Spec.Backend != nil {
//	//	repository = schedule.Spec.Backend.String()
//	//}
//	//if schedule.Spec.Archive != nil && schedule.Spec.Archive.RestoreSpec == nil {
//	//	schedule.Spec.Archive.RestoreSpec = &k8upv1.RestoreSpec{}
//	//}
//	config := job.NewConfig(ctx, r.Client, log, schedule, r.Scheme, schedule.Spec.Node)
//
//	return ctrl.Result{}, handler.NewCITAScheduleHandler(config, schedule, effectiveSchedules).Handle()
//}
//
//// SetupWithManager configures the reconciler.
//func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
//	r.Client = mgr.GetClient()
//	r.Scheme = mgr.GetScheme()
//	r.Log = l
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&citav1.Schedule{}).
//		WithEventFilter(predicate.GenerationChangedPredicate{}).
//		Complete(r)
//}
//
//// fetchEffectiveSchedules retrieves a list of EffectiveSchedules and filter the one that matches the given schedule.
//// Returns an error if the listing failed, but empty map when no matching EffectiveSchedule object was found.
//func (r *ScheduleReconciler) fetchEffectiveSchedules(ctx context.Context, schedule *citav1.Schedule) (map[k8upv1.JobType]k8upv1.EffectiveSchedule, error) {
//	list := k8upv1.EffectiveScheduleList{}
//	err := r.Client.List(ctx, &list, client.InNamespace(cfg.Config.OperatorNamespace))
//	if err != nil {
//		return map[k8upv1.JobType]k8upv1.EffectiveSchedule{}, err
//	}
//	return filterEffectiveSchedulesForReferencesOfSchedule(list, schedule), nil
//}
//
//// filterEffectiveSchedulesForReferencesOfSchedule iterates over the given list of EffectiveSchedules and returns results that reference the given schedule in their spec.
//// It returns an empty map if no element matches.
//func filterEffectiveSchedulesForReferencesOfSchedule(list k8upv1.EffectiveScheduleList, schedule *citav1.Schedule) map[k8upv1.JobType]k8upv1.EffectiveSchedule {
//	filtered := map[k8upv1.JobType]k8upv1.EffectiveSchedule{}
//	for _, es := range list.Items {
//		if es.GetDeletionTimestamp() != nil {
//			continue
//		}
//		for _, jobRef := range es.Spec.ScheduleRefs {
//			if schedule.IsReferencedBy(jobRef) {
//				filtered[es.Spec.JobType] = es
//			}
//		}
//	}
//	return filtered
//}
