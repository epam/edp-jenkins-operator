package chain

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
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
		return fmt.Errorf("failed to create jenkins client: %w", err)
	}

	codebaseTags, err := json.Marshal(jenkinsDeploy.Spec.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal codebaseTags to json: %w", err)
	}

	jobParameters := map[string]string{
		"AUTODEPLOY":       "true",
		"CODEBASE_VERSION": string(codebaseTags),
	}

	if err := jc.TriggerJob(jenkinsDeploy.Spec.Job, jobParameters); err != nil {
		return fmt.Errorf("failed to trigger job: %w", err)
	}

	log.Info("deploy job has been triggered.")

	return nextServeOrNil(h.next, jenkinsDeploy)
}

func (h TriggerJenkinsDeployJob) initJenkinsClient(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) (*jenkinsClient.JenkinsClient, error) {
	jenkinsInstance, err := h.getJenkins(jenkinsDeploy)
	if err != nil {
		return nil, fmt.Errorf("failed to get jenkins: %w", err)
	}

	jenkinsCl, err := jenkinsClient.InitGoJenkinsClient(jenkinsInstance, h.platform)
	if err != nil {
		return nil, fmt.Errorf("failed to init Jenkins Client: %w", err)
	}

	return jenkinsCl, nil
}

func (h TriggerJenkinsDeployJob) getJenkins(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) (*jenkinsApi.Jenkins, error) {
	jenkinsInstance, err := platform.GetJenkinsInstance(h.client, jenkinsDeploy.Labels[JenkinsKey], jenkinsDeploy.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get jenkins instance: %w", err)
	}

	return jenkinsInstance, nil
}
