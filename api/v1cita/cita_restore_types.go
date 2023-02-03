package v1cita

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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

func (r *RestoreSpec) CreateObject(name, namespace string) runtime.Object {
	return &Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *r,
	}
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

func (r *Restore) GetRuntimeObject() runtime.Object {
	return r
}

func (r *Restore) GetMetaObject() metav1.Object {
	return r
}

// GetJobName returns the name of the underlying batch/v1 job.
func (r *Restore) GetJobName() string {
	return r.GetType().String() + "-" + r.Name
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
	return r.Spec.KeepJobs
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (r *Restore) GetSuccessfulJobsHistoryLimit() *int {
	if r.Spec.SuccessfulJobsHistoryLimit != nil {
		return r.Spec.SuccessfulJobsHistoryLimit
	}
	return r.Spec.KeepJobs
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

// GetObjectCreator returns the ObjectCreator instance
func (in *CITARestoreSchedule) GetObjectCreator() k8upv1.ObjectCreator {
	return in
}

func (in *CITARestoreSchedule) GetNodeInfo() *NodeInfo {
	return &in.NodeInfo
}
