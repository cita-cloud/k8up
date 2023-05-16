package v1cita

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
)

// RestoreSpec can either contain an S3 restore point or a local one. For the local
// one you need to define an existing PVC.
type RestoreSpec struct {
	// inherit k8upv1.RestoreSpec
	k8upv1.RestoreSpec `json:",inline"`
	// CITACommon
	NodeInfo `json:",inline"`
	// Backup
	Backup string `json:"backup,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule Ref",type="string",JSONPath=`.metadata.ownerReferences[?(@.kind == "Schedule")].name`,description="Reference to Schedule"
// +kubebuilder:printcolumn:name="Completion",type="string",JSONPath=`.status.conditions[?(@.type == "Completed")].reason`,description="Status of Completion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Restore is the Schema for the restores API
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestoreSpec   `json:"spec,omitempty"`
	Status k8upv1.Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RestoreList contains a list of Restore
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Restore{}, &RestoreList{})
}

func (r *Restore) GetType() k8upv1.JobType {
	return CITARestoreType
}

// GetStatus retrieves the Status property
func (r *Restore) GetStatus() k8upv1.Status {
	return r.Status
}

// SetStatus sets the Status property
func (r *Restore) SetStatus(status k8upv1.Status) {
	r.Status = status
}

// GetResources returns the resource requirements
func (r *Restore) GetResources() corev1.ResourceRequirements {
	return r.Spec.Resources
}

// GetPodSecurityContext returns the pod security context
func (r *Restore) GetPodSecurityContext() *corev1.PodSecurityContext {
	return r.Spec.PodSecurityContext
}

// GetActiveDeadlineSeconds implements JobObject
func (r *Restore) GetActiveDeadlineSeconds() *int64 {
	return r.Spec.ActiveDeadlineSeconds
}

// GetFailedJobsHistoryLimit returns failed jobs history limit.
// Returns KeepJobs if unspecified.
func (r *Restore) GetFailedJobsHistoryLimit() *int {
	if r.Spec.FailedJobsHistoryLimit != nil {
		return r.Spec.FailedJobsHistoryLimit
	}
	return pointer.Int(KeepJobs)
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (r *Restore) GetSuccessfulJobsHistoryLimit() *int {
	if r.Spec.SuccessfulJobsHistoryLimit != nil {
		return r.Spec.SuccessfulJobsHistoryLimit
	}
	return pointer.Int(KeepJobs)
}

// GetJobObjects returns a sortable list of jobs
func (r *RestoreList) GetJobObjects() k8upv1.JobObjectList {
	items := make(k8upv1.JobObjectList, len(r.Items))
	for i := range r.Items {
		items[i] = &r.Items[i]
	}
	return items
}

// GetDeepCopy returns a deep copy
func (in *CITARestoreSchedule) GetDeepCopy() ScheduleSpecInterface {
	return in.DeepCopy()
}

// GetRunnableSpec returns a pointer to RunnableSpec
func (in *CITARestoreSchedule) GetRunnableSpec() *k8upv1.RunnableSpec {
	return &in.RunnableSpec
}

// GetSchedule returns the schedule definition
func (in *CITARestoreSchedule) GetSchedule() k8upv1.ScheduleDefinition {
	return in.Schedule
}

func (in *CITARestoreSchedule) GetNodeInfo() *NodeInfo {
	return &in.NodeInfo
}
