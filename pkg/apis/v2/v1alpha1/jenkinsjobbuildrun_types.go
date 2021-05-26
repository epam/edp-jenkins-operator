package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JobBuildRunStatusCreated   = "created"
	JobBuildRunStatusCompleted = "completed"
	JobBuildRunStatusFailed    = "failed"
	JobBuildRunStatusRetrying  = "retrying"
	JobBuildRunStatusNotFound  = "jobNotFound"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsJobBuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsJobBuildRunSpec   `json:"spec,omitempty"`
	Status JenkinsJobBuildRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsJobBuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsJobBuildRun `json:"items"`
}

// +k8s:openapi-gen=true
type JenkinsJobBuildRunSpec struct {
	JobPath                       string            `json:"jobpath"`
	Params                        map[string]string `json:"params"`
	Retry                         int               `json:"retry"`
	OwnerName                     *string           `json:"ownerName,omitempty"`
	DeleteAfterCompletionInterval *string           `json:"deleteAfterCompletionInterval,omitempty"`
}

func (in *JenkinsJobBuildRun) GetDeleteAfterCompletionInterval() time.Duration {
	if in.Spec.DeleteAfterCompletionInterval == nil {
		return time.Hour
	}

	dur, err := time.ParseDuration(*in.Spec.DeleteAfterCompletionInterval)
	if err != nil {
		return time.Hour
	}

	return dur
}

// +k8s:openapi-gen=true
type JenkinsJobBuildRunStatus struct {
	Status      string    `json:"status"`
	Launches    int       `json:"launches"`
	BuildNumber int64     `json:"buildNumber"`
	LastUpdated time.Time `json:"lastUpdated"`
}

func init() {
	SchemeBuilder.Register(&JenkinsJobBuildRun{}, &JenkinsJobBuildRunList{})
}
