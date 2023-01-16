package schedulecontroller

import (
	"context"
	"fmt"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"

	"github.com/go-logr/logr"
	"github.com/imdario/mergo"
	"github.com/k8up-io/k8up/v2/operator/monitoring"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/scheduler"
)

// CITAScheduleHandler handles the reconciles for the schedules. Schedules are a special
// type of k8up objects as they will only trigger jobs indirectly.
type CITAScheduleHandler struct {
	schedule *citav1.Schedule
	Log      logr.Logger
	job.Config
}

// NewCITAScheduleHandler will return a new ScheduleHandler.
func NewCITAScheduleHandler(config job.Config, schedule *citav1.Schedule, logger logr.Logger) *CITAScheduleHandler {
	return &CITAScheduleHandler{
		schedule: schedule,
		Config:   config,
		Log:      logger,
	}
}

// Handle handles the schedule management. It's responsible for adding and removing the
// jobs from the internal cron library.
func (s *CITAScheduleHandler) Handle(ctx context.Context) error {
	var err error

	err = s.createJobList(ctx)
	if err != nil {
		s.SetConditionFalseWithMessage(ctx, k8upv1.ConditionReady, k8upv1.ReasonFailed, "cannot add to cron: %v", err.Error())
		return err
	}

	if err := s.Client.Status().Update(ctx, s.schedule); err != nil {
		// Update effective schedules.
		return err
	}

	s.SetConditionTrue(ctx, k8upv1.ConditionReady, k8upv1.ReasonReady)

	_, err = controllerutil.CreateOrUpdate(ctx, s.Client, s.schedule, func() error {
		controllerutil.AddFinalizer(s.schedule, k8upv1.ScheduleFinalizerName)
		controllerutil.RemoveFinalizer(s.schedule, k8upv1.LegacyScheduleFinalizerName)
		return nil
	})

	return err
}

func (s *CITAScheduleHandler) createJobList(ctx context.Context) error {
	type objectInstantiator struct {
		spec citav1.ScheduleSpecInterface
		ctor func(spec citav1.ScheduleSpecInterface) k8upv1.JobObject
	}

	for jobType, jb := range map[k8upv1.JobType]objectInstantiator{} {
		sched := scheduler.GetScheduler()
		key := CITAKeyOf(s.schedule, jobType)
		hasSchedule := sched.HasSchedule(key)
		if k8upv1.IsNil(jb.spec) {
			if hasSchedule {
				monitoring.DecRegisteredSchedulesGauge(s.schedule.Namespace)
			}
			sched.RemoveSchedule(ctx, key)
			s.cleanupEffectiveSchedules(jobType, "")
			continue
		}
		if !hasSchedule {
			monitoring.IncRegisteredSchedulesGauge(s.schedule.Namespace)
		}
		template := jb.spec.GetDeepCopy()
		s.mergeWithDefaults(template.GetRunnableSpec())
		// CITA customize
		s.mergeWithNodeInfo(template.GetNodeInfo())
		obj := jb.ctor(template)

		s.cleanupEffectiveSchedules(jobType, template.GetSchedule())
		err := sched.SetSchedule(ctx, key, s.getEffectiveSchedule(jobType, template.GetSchedule()), func(ctx context.Context) {
			s.executeCronSchedule(ctx, obj)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func CITAKeyOf(schedule *citav1.Schedule, jobType k8upv1.JobType) string {
	key := fmt.Sprintf("%s/%s/%s", schedule.Namespace, schedule.Name, jobType)
	return key
}

func (s *CITAScheduleHandler) executeCronSchedule(ctx context.Context, obj k8upv1.JobObject) {
	obj.SetNamespace(s.schedule.Namespace)
	obj.SetName(generateName(obj.GetType(), s.schedule.Name))
	_ = controllerutil.SetOwnerReference(s.schedule, obj, s.Client.Scheme())
	log := controllerruntime.LoggerFrom(ctx)
	err := s.Client.Create(ctx, obj.DeepCopyObject().(client.Object))
	if err != nil {
		log.Error(err, "Could not create new object", "type", obj.GetType(), "namespace", obj.GetNamespace(), "name", obj.GetName())
	}
}

//func generateName(jobType k8upv1.JobType, prefix string) string {
//	lenRandom := 5
//	remainingLength := 63 - lenRandom - len(jobType) - 2
//	shortPrefix := strings.ShortenString(prefix, remainingLength)
//	return fmt.Sprintf("%s-%s-%s", shortPrefix, jobType, rand.String(lenRandom))
//}

func (s *CITAScheduleHandler) mergeWithDefaults(specInstance *k8upv1.RunnableSpec) {
	s.mergeResourcesWithDefaults(specInstance)
	s.mergeBackendWithDefaults(specInstance)
	s.mergeSecurityContextWithDefaults(specInstance)
}

func (s *CITAScheduleHandler) mergeWithNodeInfo(info *citav1.NodeInfo) {
	if info.Node == "" {
		info.Node = s.schedule.Spec.Node
	}
	if info.Chain == "" {
		info.Chain = s.schedule.Spec.Chain
	}
	if info.DeployMethod == "" {
		info.DeployMethod = s.schedule.Spec.DeployMethod
	}
}

func (s *CITAScheduleHandler) mergeResourcesWithDefaults(specInstance *k8upv1.RunnableSpec) {
	resources := &specInstance.Resources

	if err := mergo.Merge(resources, s.schedule.Spec.ResourceRequirementsTemplate); err != nil {
		s.Log.Info("could not merge specific resources with schedule defaults", "err", err.Error())
	}
	if err := mergo.Merge(resources, cfg.Config.GetGlobalDefaultResources()); err != nil {
		s.Log.Info("could not merge specific resources with global defaults", "err", err.Error())
	}
}

func (s *CITAScheduleHandler) mergeBackendWithDefaults(specInstance *k8upv1.RunnableSpec) {
	if specInstance.Backend == nil {
		specInstance.Backend = s.schedule.Spec.Backend.DeepCopy()
		return
	}

	if err := mergo.Merge(specInstance.Backend, s.schedule.Spec.Backend); err != nil {
		s.Log.Info("could not merge the schedule's backend with the resource's backend", "err", err.Error())
	}
}

func (s *CITAScheduleHandler) mergeSecurityContextWithDefaults(specInstance *k8upv1.RunnableSpec) {
	if specInstance.PodSecurityContext == nil {
		specInstance.PodSecurityContext = s.schedule.Spec.PodSecurityContext.DeepCopy()
		return
	}
	if s.schedule.Spec.PodSecurityContext == nil {
		return
	}

	if err := mergo.Merge(specInstance.PodSecurityContext, s.schedule.Spec.PodSecurityContext); err != nil {
		s.Log.Info("could not merge the schedule's security context with the resource's security context", "err", err.Error())
	}
}

func (s *CITAScheduleHandler) createRandomSchedule(jobType k8upv1.JobType, originalSchedule k8upv1.ScheduleDefinition) (k8upv1.ScheduleDefinition, error) {
	seed := createSeedForCITA(s.schedule, jobType)
	randomizedSchedule, err := randomizeSchedule(seed, originalSchedule)
	if err != nil {
		return originalSchedule, err
	}

	s.Log.V(1).Info("Randomized schedule", "seed", seed, "from_schedule", originalSchedule, "effective_schedule", randomizedSchedule)
	return randomizedSchedule, nil
}
