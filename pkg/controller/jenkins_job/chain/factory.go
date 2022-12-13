package chain

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	jobhandler "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

var log = ctrl.Log.WithName("jenkins_job")

type Chain interface {
	Build() jobhandler.JenkinsJobHandler
}

func NewChain(chain Chain) jobhandler.JenkinsJobHandler {
	return chain.Build()
}

type DefChain struct {
	client   client.Client
	platform platform.PlatformService
	log      logr.Logger
}

func InitDefChain(scheme *runtime.Scheme, k8sClient client.Client) (*DefChain, error) {
	env, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to GetPlatformTypeEnv: %w", err)
	}

	ps, err := platform.NewPlatformService(env, scheme, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewPlatformService: %w", err)
	}

	return &DefChain{
		client:   k8sClient,
		platform: ps,
		log:      log.WithName("default-chain"),
	}, nil
}

func (c DefChain) Build() jobhandler.JenkinsJobHandler {
	return PutJenkinsPipeline{
		client: c.client,
		ps:     c.platform,
		log:    c.log,
	}
}

type TriggerJobProvisionChain struct {
	client   client.Client
	platform platform.PlatformService
	log      logr.Logger
}

func InitTriggerJobProvisionChain(scheme *runtime.Scheme, k8sClient client.Client) (*TriggerJobProvisionChain, error) {
	env, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to get PlatformtypeEnv: %w", err)
	}

	ps, err := platform.NewPlatformService(env, scheme, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create new PlatformService: %w", err)
	}

	return &TriggerJobProvisionChain{
		client:   k8sClient,
		platform: ps,
		log:      log.WithName("trigger-job-provision-chain"),
	}, nil
}

func (c TriggerJobProvisionChain) Build() jobhandler.JenkinsJobHandler {
	return TriggerJobProvision{
		client: c.client,
		ps:     c.platform,
		log:    c.log,
	}
}

func nextServeOrNil(next jobhandler.JenkinsJobHandler, jj *jenkinsApi.JenkinsJob) error {
	if next == nil {
		log.Info("handling of jenkins job has been finished", "name", jj.Name)

		return nil
	}

	if err := next.ServeRequest(jj); err != nil {
		return fmt.Errorf("failed to serve next request: %w", err)
	}

	return nil
}
