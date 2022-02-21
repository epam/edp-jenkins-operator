package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type JenkinsAuthorizationRoleMappingSpec struct {
	OwnerName *string  `json:"ownerName"`
	Group     string   `json:"group"`
	RoleType  string   `json:"roleType"`
	Roles     []string `json:"roles"`
}

// +k8s:openapi-gen=true
type JenkinsAuthorizationRoleMappingStatus struct {
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsAuthorizationRoleMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsAuthorizationRoleMappingSpec   `json:"spec,omitempty"`
	Status            JenkinsAuthorizationRoleMappingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsAuthorizationRoleMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAuthorizationRoleMapping `json:"items"`
}
