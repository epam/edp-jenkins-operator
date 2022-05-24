package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JenkinsScriptSpec struct {
	// +optional
	SourceCmName string `json:"sourceConfigMapName,omitempty"`
	// +nullable
	// +optional
	OwnerName *string `json:"ownerName,omitempty"`
}

// JenkinsScriptStatus defines the observed state of JenkinsScript
type JenkinsScriptStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	Executed bool `json:"executed,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsScript is the Schema for the jenkinsscripts API
type JenkinsScript struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JenkinsScriptSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsScriptStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsScriptList contains a list of JenkinsScript
type JenkinsScriptList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsScript `json:"items"`
}
