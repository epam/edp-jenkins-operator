package chain

import (
	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

type TriggerJenkinsDeployJob struct {
	next     handler.CDStageJenkinsDeploymentHandler
	client   client.Client
	platform ps.PlatformService
	log      logr.Logger
}

const JenkinsKey = "jenkinsName"

func (h TriggerJenkinsDeployJob) ServeRequest(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) error {
	log := h.log.WithValues("job", jenkinsDeploy.Spec.Job)
	log.Info("triggering deploy job.")

	jc, err := h.initJenkinsClient(jenkinsDeploy)
	if err != nil {
		return errors.Wrap(err, "couldn't create jenkins client")
	}

	codebaseTags, err := json.Marshal(jenkinsDeploy.Spec.Tags)
	if err != nil {
		return err
	}

	jobParameters := map[string]string{
		"AUTODEPLOY":       "true",
		"CODEBASE_VERSION": string(codebaseTags),
	}

	if err := jc.TriggerJob(jenkinsDeploy.Spec.Job, jobParameters); err != nil {
		return err
	}

	log.Info("deploy job has been triggered.")
	return nextServeOrNil(h.next, jenkinsDeploy)
}
func (h TriggerJenkinsDeployJob) initJenkinsClient(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) (*jenkinsClient.JenkinsClient, error) {
	j, err := h.getJenkins(jenkinsDeploy)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get jenkins")
	}
	return jenkinsClient.InitGoJenkinsClient(j, h.platform)
}

func (h TriggerJenkinsDeployJob) getJenkins(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) (*jenkinsApi.Jenkins, error) {
	return platform.GetJenkinsInstance(h.client, jenkinsDeploy.Labels[JenkinsKey], jenkinsDeploy.Namespace)
}
