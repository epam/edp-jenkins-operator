package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	failed = "failed"
)

// CDStageJenkinsDeploymentSpec defines the desired state of CDStageJenkinsDeployment
type CDStageJenkinsDeploymentSpec struct {
	// +optional
	Job string `json:"job,omitempty"`
	// +optional
	Tag Tag `json:"tag,omitempty"`
	// +nullable
	// +optional
	Tags []Tag `json:"tags,omitempty"`
}

type Tag struct {
	Codebase string `json:"codebase"`
	Tag      string `json:"tag"`
}

// CDStageJenkinsDeploymentStatus defines the observed state of CDStageJenkinsDeploymentStatus
type CDStageJenkinsDeploymentStatus struct {
	// +optional
	Status string `json:"status,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// CDStageJenkinsDeployment is the Schema for the cdstagejenkinsdeployments API
type CDStageJenkinsDeployment struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec CDStageJenkinsDeploymentSpec `json:"spec,omitempty"`
	// +optional
	Status CDStageJenkinsDeploymentStatus `json:"status,omitempty"`
}

func (in *CDStageJenkinsDeployment) SetFailedStatus(err error) {
	in.Status.Status = failed
	in.Status.Message = err.Error()
}

//+kubebuilder:object:root=true

// CDStageJenkinsDeploymentList contains a list of CDStageJenkinsDeployment
type CDStageJenkinsDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDStageJenkinsDeployment `json:"items"`
}
