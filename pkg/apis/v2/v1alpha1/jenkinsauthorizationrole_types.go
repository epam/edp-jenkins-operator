package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type JenkinsAuthorizationRoleSpec struct {
	Name        string   `json:"name"`
	RoleType    string   `json:"roleType"`
	Permissions []string `json:"permissions"`
	Pattern     string   `json:"pattern"`
	OwnerName   *string  `json:"ownerName"`
}

// +k8s:openapi-gen=true
type JenkinsAuthorizationRoleStatus struct {
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsAuthorizationRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsAuthorizationRoleSpec   `json:"spec,omitempty"`
	Status            JenkinsAuthorizationRoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsAuthorizationRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAuthorizationRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JenkinsAuthorizationRole{}, &JenkinsAuthorizationRoleList{})
}