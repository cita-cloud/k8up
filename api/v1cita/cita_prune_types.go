package v1cita

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
)

// PruneSpec needs to contain the repository information as well as the desired
// retention policies.
type PruneSpec struct {
	// inherit k8upv1.PruneSpec
	k8upv1.PruneSpec `json:",inline"`
	// CITACommon
	NodeInfo `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule Ref",type="string",JSONPath=`.metadata.ownerReferences[?(@.kind == "Schedule")].name`,description="Reference to Schedule"
// +kubebuilder:printcolumn:name="Completion",type="string",JSONPath=`.status.conditions[?(@.type == "Completed")].reason`,description="Status of Completion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Prune is the Schema for the prunes API
type Prune struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PruneSpec     `json:"spec,omitempty"`
	Status k8upv1.Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PruneList contains a list of Prune
type PruneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Prune `json:"items"`
}

func (p *Prune) GetType() k8upv1.JobType {
	return CITAPruneType
}

// GetStatus retrieves the Status property
func (p *Prune) GetStatus() k8upv1.Status {
	return p.Status
}

// SetStatus sets the Status property
func (p *Prune) SetStatus(status k8upv1.Status) {
	p.Status = status
}

// GetResources returns the resource requirements
func (p *Prune) GetResources() corev1.ResourceRequirements {
	return p.Spec.Resources
}

// GetPodSecurityContext returns the pod security context
func (p *Prune) GetPodSecurityContext() *corev1.PodSecurityContext {
	return p.Spec.PodSecurityContext
}

// GetActiveDeadlineSeconds implements JobObject
func (p *Prune) GetActiveDeadlineSeconds() *int64 {
	return p.Spec.ActiveDeadlineSeconds
}

// GetFailedJobsHistoryLimit returns failed jobs history limit.
// Returns KeepJobs if unspecified.
func (p *Prune) GetFailedJobsHistoryLimit() *int {
	if p.Spec.FailedJobsHistoryLimit != nil {
		return p.Spec.FailedJobsHistoryLimit
	}
	return p.Spec.KeepJobs
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (p *Prune) GetSuccessfulJobsHistoryLimit() *int {
	if p.Spec.SuccessfulJobsHistoryLimit != nil {
		return p.Spec.SuccessfulJobsHistoryLimit
	}
	return p.Spec.KeepJobs
}

// GetJobObjects returns a sortable list of jobs
func (p *PruneList) GetJobObjects() k8upv1.JobObjectList {
	items := make(k8upv1.JobObjectList, len(p.Items))
	for i := range p.Items {
		items[i] = &p.Items[i]
	}
	return items
}

// GetDeepCopy returns a deep copy
func (in *CITAPruneSchedule) GetDeepCopy() ScheduleSpecInterface {
	return in.DeepCopy()
}

// GetRunnableSpec returns a pointer to RunnableSpec
func (in *CITAPruneSchedule) GetRunnableSpec() *k8upv1.RunnableSpec {
	return &in.RunnableSpec
}

// GetSchedule returns the schedule definition
func (in *CITAPruneSchedule) GetSchedule() k8upv1.ScheduleDefinition {
	return in.Schedule
}

func (in *CITAPruneSchedule) GetNodeInfo() *NodeInfo {
	return &in.NodeInfo
}

func init() {
	SchemeBuilder.Register(&Prune{}, &PruneList{})
}
