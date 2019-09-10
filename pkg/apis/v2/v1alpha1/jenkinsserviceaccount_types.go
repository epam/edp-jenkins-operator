package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsServiceAccountSpec defines the desired state of JenkinsServiceAccount
// +k8s:openapi-gen=true
type JenkinsServiceAccountSpec struct {
	Type        string `json:"type"`
	Credentials string `json:"credentials"`
	OwnerName   string `json:"ownerName, omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// JenkinsServiceAccountStatus defines the observed state of JenkinsServiceAccount
// +k8s:openapi-gen=true
type JenkinsServiceAccountStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool      `json:"available, omitempty"`
	Created         bool      `json:"created, omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated, omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsServiceAccount is the Schema for the jenkinsserviceaccounts API
// +k8s:openapi-gen=true
type JenkinsServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsServiceAccountSpec   `json:"spec,omitempty"`
	Status JenkinsServiceAccountStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsServiceAccountList contains a list of JenkinsServiceAccount
type JenkinsServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsServiceAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsServiceAccount{}, &JenkinsServiceAccountList{})
}
