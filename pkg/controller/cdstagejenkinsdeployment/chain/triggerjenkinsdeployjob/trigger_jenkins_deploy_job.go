package triggerjenkinsdeployjob

import (
	"encoding/json"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TriggerJenkinsDeployJob struct {
	Client   client.Client
	Platform ps.PlatformService
	Log      logr.Logger
}

const jenkinsKey = "jenkinsName"

func (h TriggerJenkinsDeployJob) ServeRequest(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) error {
	log := h.Log.WithValues("job", jenkinsDeploy.Spec.Job)
	log.Info("triggering deploy job.")

	jc, err := h.initJenkinsClient(jenkinsDeploy)
	if err != nil {
		return errors.Wrap(err, "couldn't create jenkins client")
	}

	codebases, err := json.Marshal(jenkinsDeploy.Spec.Tags)
	if err != nil {
		return err
	}

	jobParameters := map[string]string{
		"AUTODEPLOY":        "true",
		"CODEBASE_VERSIONS": string(codebases),
	}

	if err := jc.TriggerJob(jenkinsDeploy.Spec.Job, jobParameters); err != nil {
		return errors.Wrapf(err, "couldn't trigger jenkins job %v", jenkinsDeploy.Spec.Job)
	}

	log.Info("deploy job has been triggered.")
	return nil
}
func (h TriggerJenkinsDeployJob) initJenkinsClient(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) (*jenkinsClient.JenkinsClient, error) {
	j, err := h.getJenkins(jenkinsDeploy)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get jenkins")
	}
	return jenkinsClient.InitGoJenkinsClient(j, h.Platform)
}

func (h TriggerJenkinsDeployJob) getJenkins(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) (*v1alpha1.Jenkins, error) {
	return platform.GetJenkinsInstance(h.Client, jenkinsDeploy.Labels[jenkinsKey], jenkinsDeploy.Namespace)
}
