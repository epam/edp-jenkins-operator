package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type JenkinsAuthorizationRoleMappingSpec struct {
	// +nullable
	// +optional
	OwnerName *string  `json:"ownerName,omitempty"`
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
// +kubebuilder:deprecatedversion
type JenkinsAuthorizationRoleMapping struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec JenkinsAuthorizationRoleMappingSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsAuthorizationRoleMappingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsAuthorizationRoleMappingList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAuthorizationRoleMapping `json:"items"`
}
