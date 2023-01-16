package handler

import (
	"fmt"

	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	citav1 "github.com/k8up-io/k8up/v2/api/v1cita"
	"github.com/k8up-io/k8up/v2/operator/cfg"
	"github.com/k8up-io/k8up/v2/operator/job"
	"github.com/k8up-io/k8up/v2/operator/scheduler"
)

// CITAScheduleHandler handles the reconciles for the schedules. Schedules are a special
// type of k8up objects as they will only trigger jobs indirectly.
type CITAScheduleHandler struct {
	schedule           *citav1.Schedule
	effectiveSchedules map[k8upv1.JobType]k8upv1.EffectiveSchedule
	job.Config
	requireStatusUpdate bool
}

// NewCITAScheduleHandler will return a new ScheduleHandler.
func NewCITAScheduleHandler(
	config job.Config, schedule *citav1.Schedule,
	effectiveSchedules map[k8upv1.JobType]k8upv1.EffectiveSchedule) *CITAScheduleHandler {

	return &CITAScheduleHandler{
		schedule:           schedule,
		effectiveSchedules: effectiveSchedules,
		Config:             config,
	}
}

// Handle handles the schedule management. It's responsible for adding and removing the
// jobs from the internal cron library.
func (s *CITAScheduleHandler) Handle() error {

	if s.schedule.GetDeletionTimestamp() != nil {
		return s.finalizeSchedule()
	}

	var err error

	jobList := s.createJobList()

	err = scheduler.GetScheduler().SyncSchedules(jobList)
	if err != nil {
		s.SetConditionFalseWithMessage(k8upv1.ConditionReady, k8upv1.ReasonFailed, "cannot add to cron: %v", err.Error())
		return err
	}

	if err := s.synchronizeEffectiveSchedulesResources(); err != nil {
		// at this point, conditions are already set and updated.
		return err
	}

	s.SetConditionTrue(k8upv1.ConditionReady, k8upv1.ReasonReady)

	if controllerutil.ContainsFinalizer(s.schedule, k8upv1.LegacyScheduleFinalizerName) {
		controllerutil.AddFinalizer(s.schedule, k8upv1.ScheduleFinalizerName)
		controllerutil.RemoveFinalizer(s.schedule, k8upv1.LegacyScheduleFinalizerName)
		return s.updateSchedule()
	}

	if !controllerutil.ContainsFinalizer(s.schedule, k8upv1.ScheduleFinalizerName) {
		controllerutil.AddFinalizer(s.schedule, k8upv1.ScheduleFinalizerName)
		return s.updateSchedule()
	}
	return nil
}

func (s *CITAScheduleHandler) createJobList() scheduler.JobList {
	jobList := scheduler.JobList{
		Config: s.Config,
		Jobs:   make([]scheduler.Job, 0),
	}

	for jobType, jb := range map[k8upv1.JobType]citav1.ScheduleSpecInterface{
		citav1.CITAPruneType:   s.schedule.Spec.CITAPrune,
		citav1.CITABackupType:  s.schedule.Spec.CITABackup,
		citav1.CITARestoreType: s.schedule.Spec.CITARestore,
	} {
		if k8upv1.IsNil(jb) {
			s.cleanupEffectiveSchedules(jobType, "")
			continue
		}
		template := jb.GetDeepCopy()
		s.mergeWithDefaults(template.GetRunnableSpec())
		// CITA customize
		s.mergeWithNodeInfo(template.GetNodeInfo())
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, template.GetSchedule()),
			Object:   template.GetObjectCreator(),
		})
		s.cleanupEffectiveSchedules(jobType, template.GetSchedule())
	}

	return jobList
}

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
		s.Log.Info("could not merge specific resources with schedule defaults", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
	if err := mergo.Merge(resources, cfg.Config.GetGlobalDefaultResources()); err != nil {
		s.Log.Info("could not merge specific resources with global defaults", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
}

func (s *CITAScheduleHandler) mergeBackendWithDefaults(specInstance *k8upv1.RunnableSpec) {
	if specInstance.Backend == nil {
		specInstance.Backend = s.schedule.Spec.Backend.DeepCopy()
		return
	}

	if err := mergo.Merge(specInstance.Backend, s.schedule.Spec.Backend); err != nil {
		s.Log.Info("could not merge the schedule's backend with the resource's backend", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
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
		s.Log.Info("could not merge the schedule's security context with the resource's security context", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
}

func (s *CITAScheduleHandler) updateSchedule() error {
	if err := s.Client.Update(s.CTX, s.schedule); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error updating resource %s/%s: %w", s.schedule.Namespace, s.schedule.Name, err)
	}
	return nil
}

func (s *CITAScheduleHandler) createRandomSchedule(jobType k8upv1.JobType, originalSchedule k8upv1.ScheduleDefinition) (k8upv1.ScheduleDefinition, error) {
	seed := s.createSeed(s.schedule, jobType)
	randomizedSchedule, err := randomizeSchedule(seed, originalSchedule)
	if err != nil {
		return originalSchedule, err
	}

	s.Log.V(1).Info("Randomized schedule", "seed", seed, "from_schedule", originalSchedule, "effective_schedule", randomizedSchedule)
	return randomizedSchedule, nil
}

// finalizeSchedule ensures that all associated resources are cleaned up.
// It also removes the schedule definitions from internal scheduler.
func (s *CITAScheduleHandler) finalizeSchedule() error {
	namespacedName := k8upv1.MapToNamespacedName(s.schedule)
	controllerutil.RemoveFinalizer(s.schedule, k8upv1.ScheduleFinalizerName)
	scheduler.GetScheduler().RemoveSchedules(namespacedName)
	for jobType := range s.effectiveSchedules {
		s.cleanupEffectiveSchedules(jobType, "")
	}
	if err := s.synchronizeEffectiveSchedulesResources(); err != nil {
		return err
	}
	return s.updateSchedule()
}
