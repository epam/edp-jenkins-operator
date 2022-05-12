package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JenkinsAgentSpec struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

func (in JenkinsAgentSpec) SalvesKey() string {
	return fmt.Sprintf("%s-template", in.Name)
}

type JenkinsAgentStatus struct {
	Value string `json:"value"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
type JenkinsAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JenkinsAgentSpec   `json:"spec,omitempty"`
	Status            JenkinsAgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
type JenkinsAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsAgent `json:"items"`
}
