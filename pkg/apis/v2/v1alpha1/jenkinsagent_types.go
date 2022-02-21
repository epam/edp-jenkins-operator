package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true
type JenkinsAgentSpec struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

func (in JenkinsAgentSpec) SalvesKey() string {
	return fmt.Sprintf("%s-template", in.Name)
}

// +k8s:openapi-gen=true
type JenkinsAgentStatus struct {
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JenkinsAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsAgentSpec   `json:"spec,omitempty"`
	Status            JenkinsAgentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JenkinsAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAgent `json:"items"`
}
