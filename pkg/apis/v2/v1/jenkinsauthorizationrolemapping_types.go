package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type JenkinsAuthorizationRoleMappingSpec struct {
	OwnerName *string  `json:"ownerName,omitempty"`
	Group     string   `json:"group"`
	RoleType  string   `json:"roleType"`
	Roles     []string `json:"roles"`
}

type JenkinsAuthorizationRoleMappingStatus struct {
	Value string `json:"value"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
type JenkinsAuthorizationRoleMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsAuthorizationRoleMappingSpec   `json:"spec,omitempty"`
	Status            JenkinsAuthorizationRoleMappingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
type JenkinsAuthorizationRoleMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAuthorizationRoleMapping `json:"items"`
}
