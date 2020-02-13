package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsJobSpec defines the desired state of Jenkins job
// +k8s:openapi-gen=true

type JenkinsJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	OwnerName     *string `json:"ownerName,omitempty"`
	StageName     *string `json:"stageName,omitempty"`
	JenkinsFolder *string `json:"jenkinsFolder,omitempty"`
	Job           Job     `json:"job"`
}

type Job struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

type ActionType string
type Result string

const (
	PlatformProjectCreation ActionType = "platform_project_creation"
	RoleBinding             ActionType = "role_binding"
	CreateJenkinsPipeline   ActionType = "create_jenkins_pipeline"

	Success Result = "success"
	Error   Result = "error"
)

// JenkinsFolderStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool       `json:"available, omitempty"`
	LastTimeUpdated time.Time  `json:"lastTimeUpdated, omitempty"`
	Status          string     `json:"status, omitempty"`
	Username        string     `json:"username"`
	Action          ActionType `json:"action"`
	Result          Result     `json:"result"`
	DetailedMessage string     `json:"detailedMessage"`
	Value           string     `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsFolder is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsJobSpec   `json:"spec,omitempty"`
	Status JenkinsJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsFolderList contains a list of Jenkins
type JenkinsJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsJob{}, &JenkinsJobList{})
}
