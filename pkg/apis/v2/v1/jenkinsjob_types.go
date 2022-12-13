package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ActionType string
type Result string

const (
	RoleBinding           ActionType = "role_binding"
	CreateJenkinsPipeline ActionType = "create_jenkins_pipeline"
	TriggerJobProvision   ActionType = "trigger_job_provision"

	Success Result = "success"
	Error   Result = "error"
)

type JenkinsJobSpec struct {
	// +nullable
	// +optional
	OwnerName *string `json:"ownerName,omitempty"`
	// +nullable
	// +optional
	StageName *string `json:"stageName,omitempty"`
	// +nullable
	// +optional
	JenkinsFolder *string `json:"jenkinsFolder,omitempty"`
	Job           Job     `json:"job"`
}

type Job struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	// +nullable
	// +optional
	AutoTriggerPeriod *int32 `json:"autoTriggerPeriod,omitempty"`
}

// JenkinsJobStatus defines the observed state of JenkinsJob.
type JenkinsJobStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	// +optional
	Status          string     `json:"status,omitempty"`
	Username        string     `json:"username"`
	Action          ActionType `json:"action"`
	Result          Result     `json:"result"`
	DetailedMessage string     `json:"detailedMessage"`
	Value           string     `json:"value"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsJob is the Schema for the jenkinsjob API.
type JenkinsJob struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JenkinsJobSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsJobStatus `json:"status,omitempty"`
}

func (jj *JenkinsJob) IsAutoTriggerEnabled() bool {
	period := jj.Spec.Job.AutoTriggerPeriod

	if period == nil || *period == 0 {
		return false
	}

	var (
		minPeriod int32 = 5
		maxPeriod int32 = 7200
	)

	if *period < minPeriod || *period > maxPeriod {
		ctrl.Log.WithName("jenkins-job-api").Info("autoTriggerPeriod value is incorrect. disable auto trigger",
			"value", *period)

		return false
	}

	return true
}

//+kubebuilder:object:root=true

// JenkinsJobList contains a list of JenkinsJob.
type JenkinsJobList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsJob `json:"items"`
}
