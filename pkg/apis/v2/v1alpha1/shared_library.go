package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type JenkinsSharedLibrarySpec struct {
	OwnerName    *string `json:"ownerName,omitempty"`
	Name         string  `json:"name"`
	CredentialID string  `json:"secret"`
	Tag          string  `json:"tag,omitempty"`
	URL          string  `json:"url"`
}

type JenkinsSharedLibraryStatus struct {
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:deprecatedversion
type JenkinsSharedLibrary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsSharedLibrarySpec   `json:"spec,omitempty"`
	Status            JenkinsSharedLibraryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsSharedLibraryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsSharedLibrary `json:"items"`
}
