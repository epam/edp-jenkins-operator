package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type JenkinsSharedLibrarySpec struct {
	// +nullable
	// +optional
	OwnerName *string `json:"ownerName,omitempty"`
	Name      string  `json:"name"`
	// +optional
	CredentialID string `json:"secret,omitempty"`
	Tag          string `json:"tag"`
	URL          string `json:"url"`
}

type JenkinsSharedLibraryStatus struct {
	Value string `json:"value"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
type JenkinsSharedLibrary struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec JenkinsSharedLibrarySpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsSharedLibraryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type JenkinsSharedLibraryList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsSharedLibrary `json:"items"`
}
