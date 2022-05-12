package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JenkinsScriptSpec struct {
	SourceCmName string  `json:"sourceConfigMapName,omitempty"`
	OwnerName    *string `json:"ownerName,omitempty"`
}

// JenkinsScriptStatus defines the observed state of JenkinsScript
type JenkinsScriptStatus struct {
	Available       bool        `json:"available,omitempty"`
	Executed        bool        `json:"executed,omitempty"`
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsScript is the Schema for the jenkinsscripts API
type JenkinsScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsScriptSpec   `json:"spec,omitempty"`
	Status JenkinsScriptStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsScriptList contains a list of JenkinsScript
type JenkinsScriptList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsScript `json:"items"`
}
