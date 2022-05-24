package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsServiceAccountSpec defines the desired state of JenkinsServiceAccount
type JenkinsServiceAccountSpec struct {
	Type        string `json:"type"`
	Credentials string `json:"credentials"`
	// +optional
	OwnerName string `json:"ownerName,omitempty"`
}

// JenkinsServiceAccountStatus defines the observed state of JenkinsServiceAccount
type JenkinsServiceAccountStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	Created bool `json:"created,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsServiceAccount is the Schema for the jenkinsserviceaccounts API
type JenkinsServiceAccount struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JenkinsServiceAccountSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsServiceAccountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsServiceAccountList contains a list of JenkinsServiceAccount
type JenkinsServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsServiceAccount `json:"items"`
}
