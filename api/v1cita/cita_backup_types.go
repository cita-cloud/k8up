package v1cita

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
)

// BackupSpec defines a single backup. It must contain all information to connect to
// the backup repository when applied. If used with defaults or schedules the operator will
// ensure that the defaults are applied before creating the object on the API.
type BackupSpec struct {
	// inherit k8upv1.BackupSpec
	k8upv1.BackupSpec `json:",inline"`
	// CITACommon
	NodeInfo `json:",inline"`
	// DataType
	DataType *DataType `json:"dataType,omitempty"`
}

type DataType struct {
	Full  *FullType  `json:"full,omitempty"`
	State *StateType `json:"state,omitempty"`
}

type FullType struct {
	IncludePaths []string `json:"includePaths,omitempty"`
}

type StateType struct {
	BlockHeight int64 `json:"blockHeight,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule Ref",type="string",JSONPath=`.metadata.ownerReferences[?(@.kind == "Schedule")].name`,description="Reference to Schedule"
// +kubebuilder:printcolumn:name="Completion",type="string",JSONPath=`.status.conditions[?(@.type == "Completed")].reason`,description="Status of Completion"
// +kubebuilder:printcolumn:name="PreBackup",type="string",JSONPath=`.status.conditions[?(@.type == "PreBackupPodReady")].reason`,description="Status of PreBackupPods"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec    `json:"spec,omitempty"`
	Status k8upv1.Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}

func (*Backup) GetType() k8upv1.JobType {
	return CITABackupType
}

// GetStatus retrieves the Status property
func (b *Backup) GetStatus() k8upv1.Status {
	return b.Status
}

// SetStatus sets the Status property
func (b *Backup) SetStatus(status k8upv1.Status) {
	b.Status = status
}

// GetResources returns the resource requirements
func (b *Backup) GetResources() corev1.ResourceRequirements {
	return b.Spec.Resources
}

// GetPodSecurityContext returns the pod security context
func (b *Backup) GetPodSecurityContext() *corev1.PodSecurityContext {
	return b.Spec.PodSecurityContext
}

// GetActiveDeadlineSeconds implements JobObject
func (b *Backup) GetActiveDeadlineSeconds() *int64 {
	return b.Spec.ActiveDeadlineSeconds
}

// GetFailedJobsHistoryLimit returns failed jobs history limit.
// Returns KeepJobs if unspecified.
func (b *Backup) GetFailedJobsHistoryLimit() *int {
	if b.Spec.FailedJobsHistoryLimit != nil {
		return b.Spec.FailedJobsHistoryLimit
	}
	return b.Spec.KeepJobs
}

// GetSuccessfulJobsHistoryLimit returns successful jobs history limit.
// Returns KeepJobs if unspecified.
func (b *Backup) GetSuccessfulJobsHistoryLimit() *int {
	if b.Spec.SuccessfulJobsHistoryLimit != nil {
		return b.Spec.SuccessfulJobsHistoryLimit
	}
	return b.Spec.KeepJobs
}

// GetJobObjects returns a sortable list of jobs
func (b *BackupList) GetJobObjects() k8upv1.JobObjectList {
	items := make(k8upv1.JobObjectList, len(b.Items))
	for i := range b.Items {
		items[i] = &b.Items[i]
	}
	return items
}

// GetDeepCopy returns a deep copy
func (in *CITABackupSchedule) GetDeepCopy() ScheduleSpecInterface {
	return in.DeepCopy()
}

// GetRunnableSpec returns a pointer to RunnableSpec
func (in *CITABackupSchedule) GetRunnableSpec() *k8upv1.RunnableSpec {
	return &in.RunnableSpec
}

// GetSchedule returns the schedule definition
func (in *CITABackupSchedule) GetSchedule() k8upv1.ScheduleDefinition {
	return in.Schedule
}

func (in *CITABackupSchedule) GetNodeInfo() *NodeInfo {
	return &in.NodeInfo
}
