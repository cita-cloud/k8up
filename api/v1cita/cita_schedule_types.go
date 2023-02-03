package v1cita

import (
	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ScheduleSpec defines the schedules for the various job types.
type ScheduleSpec struct {
	// inherit k8upv1.PruneSpec
	k8upv1.ScheduleSpec `json:",inline"`

	CITARestore *CITARestoreSchedule `json:"citaRestore,omitempty"`
	CITABackup  *CITABackupSchedule  `json:"citaBackup,omitempty"`
	CITAPrune   *CITAPruneSchedule   `json:"citaPrune,omitempty"`

	// CITACommon
	NodeInfo `json:",inline"`
}

// CITARestoreSchedule manages schedules for the restore service
type CITARestoreSchedule struct {
	RestoreSpec            `json:",inline"`
	*k8upv1.ScheduleCommon `json:",inline"`
}

// CITABackupSchedule manages schedules for the backup service
type CITABackupSchedule struct {
	BackupSpec             `json:",inline"`
	*k8upv1.ScheduleCommon `json:",inline"`
}

// CITAPruneSchedule manages the schedules for the prunes
type CITAPruneSchedule struct {
	PruneSpec              `json:",inline"`
	*k8upv1.ScheduleCommon `json:",inline"`
}

//func (in *CITAPruneSchedule) GetDeepCopy() k8upv1.ScheduleSpecInterface {
//	//TODO implement me
//	panic("implement me")
//}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Schedule is the Schema for the schedules API
type Schedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduleSpec          `json:"spec,omitempty"`
	Status k8upv1.ScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScheduleList contains a list of Schedule
type ScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Schedule{}, &ScheduleList{})
}

func (s *Schedule) GetRuntimeObject() runtime.Object {
	return s
}

func (s *Schedule) GetMetaObject() metav1.Object {
	return s
}

// GetJobName implements the JobObject interface.
func (s *Schedule) GetJobName() string {
	return s.GetType().String() + "-" + s.Name
}

func (*Schedule) GetType() k8upv1.JobType {
	return CITAScheduleType
}

// GetStatus retrieves the Status property
func (s *Schedule) GetStatus() k8upv1.Status {
	return k8upv1.Status{Conditions: s.Status.Conditions}
}

// SetStatus sets the Status.Conditions property
func (s *Schedule) SetStatus(status k8upv1.Status) {
	s.Status.Conditions = status.Conditions
}

// GetResources returns the resource requirements
func (s *Schedule) GetResources() corev1.ResourceRequirements {
	return s.Spec.ResourceRequirementsTemplate
}

// GetPodSecurityContext returns the pod security context
func (s *Schedule) GetPodSecurityContext() *corev1.PodSecurityContext {
	return s.Spec.PodSecurityContext
}

// GetActiveDeadlineSeconds implements JobObject
func (s *Schedule) GetActiveDeadlineSeconds() *int64 {
	return nil
}

// GetFailedJobsHistoryLimit returns failed jobs history limit.
// Returns KeepJobs if unspecified.
func (s *Schedule) GetFailedJobsHistoryLimit() *int {
	if s.Spec.FailedJobsHistoryLimit != nil {
		return s.Spec.FailedJobsHistoryLimit
	}
	return s.Spec.KeepJobs
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (s *Schedule) GetSuccessfulJobsHistoryLimit() *int {
	if s.Spec.SuccessfulJobsHistoryLimit != nil {
		return s.Spec.SuccessfulJobsHistoryLimit
	}
	return s.Spec.KeepJobs
}

// IsReferencedBy returns true if the given ref matches the schedule's name and namespace.
func (s *Schedule) IsReferencedBy(ref k8upv1.ScheduleRef) bool {
	return ref.Namespace == s.Namespace && ref.Name == s.Name
}
