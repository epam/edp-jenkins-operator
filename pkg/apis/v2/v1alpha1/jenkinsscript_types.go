package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsScriptSpec defines the desired state of JenkinsScript
// +k8s:openapi-gen=true
type JenkinsScriptSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	SourceCmName string `json:"source_cm_name, omitempty"`
}

// JenkinsScriptStatus defines the observed state of JenkinsScript
// +k8s:openapi-gen=true
type JenkinsScriptStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Executed bool `json:"available, omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsScript is the Schema for the jenkinsscripts API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsScriptSpec   `json:"spec,omitempty"`
	Status JenkinsScriptStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsScriptList contains a list of JenkinsScript
type JenkinsScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsScript `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsScript{}, &JenkinsScriptList{})
}
