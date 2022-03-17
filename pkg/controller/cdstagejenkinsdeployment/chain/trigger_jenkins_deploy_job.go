package chain

import (
	"encoding/json"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TriggerJenkinsDeployJob struct {
	next     handler.CDStageJenkinsDeploymentHandler
	client   client.Client
	platform ps.PlatformService
	log      logr.Logger
}

const JenkinsKey = "jenkinsName"

func (h TriggerJenkinsDeployJob) ServeRequest(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) error {
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
func (h TriggerJenkinsDeployJob) initJenkinsClient(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) (*jenkinsClient.JenkinsClient, error) {
	j, err := h.getJenkins(jenkinsDeploy)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get jenkins")
	}
	return jenkinsClient.InitGoJenkinsClient(j, h.platform)
}

func (h TriggerJenkinsDeployJob) getJenkins(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) (*v1alpha1.Jenkins, error) {
	return platform.GetJenkinsInstance(h.client, jenkinsDeploy.Labels[JenkinsKey], jenkinsDeploy.Namespace)
}
