package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsFolderSpec defines the desired state of JenkinsFolder
type JenkinsFolderSpec struct {
	// +optional
	// +nullable
	CodebaseName *string `json:"codebaseName,omitempty"`
	// +optional
	// +nullable
	OwnerName *string `json:"ownerName,omitempty"`
	// +nullable
	// +optional
	Job *Job `json:"job"`
}

// JenkinsFolderStatus defines the observed state of JenkinsFolder
type JenkinsFolderStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	// +optional
	Status string `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsFolder is the Schema for the jenkinsfolder API
type JenkinsFolder struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JenkinsFolderSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsFolderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsFolderList contains a list of JenkinsFolder
type JenkinsFolderList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsFolder `json:"items"`
}
