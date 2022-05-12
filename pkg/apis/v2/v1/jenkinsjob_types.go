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
	OwnerName     *string `json:"ownerName,omitempty"`
	StageName     *string `json:"stageName,omitempty"`
	JenkinsFolder *string `json:"jenkinsFolder,omitempty"`
	Job           Job     `json:"job"`
}

type Job struct {
	Name              string `json:"name"`
	Config            string `json:"config"`
	AutoTriggerPeriod *int32 `json:"autoTriggerPeriod,omitempty"`
}

// JenkinsJobStatus defines the observed state of JenkinsJob
type JenkinsJobStatus struct {
	Available                      bool        `json:"available,omitempty"`
	LastTimeUpdated                metav1.Time `json:"lastTimeUpdated,omitempty"`
	Status                         string      `json:"status,omitempty"`
	JenkinsJobProvisionBuildNumber int64       `json:"jenkinsJobProvisionBuildNumber"`
	Username                       string      `json:"username"`
	Action                         ActionType  `json:"action"`
	Result                         Result      `json:"result"`
	DetailedMessage                string      `json:"detailedMessage"`
	Value                          string      `json:"value"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// JenkinsJob is the Schema for the jenkinsjob API
type JenkinsJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsJobSpec   `json:"spec,omitempty"`
	Status JenkinsJobStatus `json:"status,omitempty"`
}

func (jj JenkinsJob) IsAutoTriggerEnabled() bool {
	period := jj.Spec.Job.AutoTriggerPeriod
	if period == nil || *period == 0 {
		return false
	}
	if *period < 5 || *period > 7200 {
		ctrl.Log.WithName("jenkins-job-api").Info("autoTriggerPeriod value is incorrect. disable auto trigger",
			"value", *period)
		return false
	}
	return true
}

//+kubebuilder:object:root=true

// JenkinsJobList contains a list of JenkinsJob
type JenkinsJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JenkinsJob `json:"items"`
}
