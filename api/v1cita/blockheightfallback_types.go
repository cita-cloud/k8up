package v1cita

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
)

type BlockHeightFallbackSpec struct {
	k8upv1.RunnableSpec `json:",inline"`

	K8upCommon `json:",inline"`
	// CITACommon
	NodeInfo `json:",inline"`
	// BlockHeight
	BlockHeight int64 `json:"blockHeight"`
	// DeleteConsensusData weather or not delete consensus data when restore
	DeleteConsensusData bool `json:"deleteConsensusData,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=bhf

// BlockHeightFallback is the Schema for the blockheightfallbacks API
type BlockHeightFallback struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlockHeightFallbackSpec `json:"spec,omitempty"`
	Status k8upv1.Status           `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BlockHeightFallbackList contains a list of BlockHeightFallback
type BlockHeightFallbackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlockHeightFallback `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlockHeightFallback{}, &BlockHeightFallbackList{})
}

func (*BlockHeightFallback) GetType() k8upv1.JobType {
	return FallbackType
}

// GetStatus retrieves the Status property
func (b *BlockHeightFallback) GetStatus() k8upv1.Status {
	return b.Status
}

// SetStatus sets the Status property
func (b *BlockHeightFallback) SetStatus(status k8upv1.Status) {
	b.Status = status
}

// GetResources returns the resource requirements
func (b *BlockHeightFallback) GetResources() corev1.ResourceRequirements {
	return b.Spec.Resources
}

// GetPodSecurityContext returns the pod security context
func (b *BlockHeightFallback) GetPodSecurityContext() *corev1.PodSecurityContext {
	return b.Spec.PodSecurityContext
}

// GetActiveDeadlineSeconds implements JobObject
func (b *BlockHeightFallback) GetActiveDeadlineSeconds() *int64 {
	return b.Spec.ActiveDeadlineSeconds
}

// GetFailedJobsHistoryLimit returns failed jobs history limit.
// Returns KeepJobs if unspecified.
func (b *BlockHeightFallback) GetFailedJobsHistoryLimit() *int {
	if b.Spec.FailedJobsHistoryLimit != nil {
		return b.Spec.FailedJobsHistoryLimit
	}
	return pointer.Int(KeepJobs)
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (b *BlockHeightFallback) GetSuccessfulJobsHistoryLimit() *int {
	if b.Spec.SuccessfulJobsHistoryLimit != nil {
		return b.Spec.SuccessfulJobsHistoryLimit
	}
	return pointer.Int(KeepJobs)
}

// GetJobObjects returns a sortable list of jobs
func (b *BlockHeightFallbackList) GetJobObjects() k8upv1.JobObjectList {
	items := make(k8upv1.JobObjectList, len(b.Items))
	for i := range b.Items {
		items[i] = &b.Items[i]
	}
	return items
}
