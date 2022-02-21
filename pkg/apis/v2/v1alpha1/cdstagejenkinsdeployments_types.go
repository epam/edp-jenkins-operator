package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	failed = "failed"
)

// CDStageJenkinsDeploymentSpec defines the desired state of CDStageJenkinsDeployment
// +k8s:openapi-gen=true

type CDStageJenkinsDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Job  string `json:"job"`
	Tag  Tag    `json:"tag"`
	Tags []Tag  `json:"tags"`
}

type Tag struct {
	Codebase string `json:"codebase"`
	Tag      string `json:"tag"`
}

// CDStageJenkinsDeploymentStatus defines the observed state of CDStageJenkinsDeploymentStatus
// +k8s:openapi-gen=true
type CDStageJenkinsDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Status       string `json:"status"`
	Message      string `json:"message"`
	FailureCount int64  `json:"failureCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDStageJenkinsDeployment is the Schema for the cdstagejenkinsdeployments API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type CDStageJenkinsDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDStageJenkinsDeploymentSpec   `json:"spec,omitempty"`
	Status CDStageJenkinsDeploymentStatus `json:"status,omitempty"`
}

func (in *CDStageJenkinsDeployment) SetFailedStatus(err error) {
	in.Status.Status = failed
	in.Status.Message = err.Error()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDStageJenkinsDeploymentList contains a list of CDStageJenkinsDeployment
type CDStageJenkinsDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDStageJenkinsDeployment `json:"items"`
}
