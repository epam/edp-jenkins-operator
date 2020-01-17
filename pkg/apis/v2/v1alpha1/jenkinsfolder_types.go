package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsSpec defines the desired state of Jenkins
// +k8s:openapi-gen=true

type JenkinsFolderSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	CodebaseName *string `json:"codebaseName"`
	OwnerName    *string `json:"ownerName"`
	JobName      *string `json:"jobName"`
}

// JenkinsFolderStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsFolderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Available                      bool      `json:"available, omitempty"`
	LastTimeUpdated                time.Time `json:"lastTimeUpdated, omitempty"`
	Status                         string    `json:"status, omitempty"`
	JenkinsJobProvisionBuildNumber int64     `json:"jenkinsJobProvisionBuildNumber"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsFolder is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsFolder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsFolderSpec   `json:"spec,omitempty"`
	Status JenkinsFolderStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsFolderList contains a list of Jenkins
type JenkinsFolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsFolder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsFolder{}, &JenkinsFolderList{})
}
