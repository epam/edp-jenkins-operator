package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	jobhandler "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func InitDefChain(scheme *runtime.Scheme, client client.Client) (*DefChain, error) {
	ps, err := platform.NewPlatformService(helper.GetPlatformTypeEnv(), scheme, &client)
	if err != nil {
		return nil, err
	}

	return &DefChain{
		client:   client,
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

func InitTriggerJobProvisionChain(scheme *runtime.Scheme, client client.Client) (*TriggerJobProvisionChain, error) {
	ps, err := platform.NewPlatformService(helper.GetPlatformTypeEnv(), scheme, &client)
	if err != nil {
		return nil, err
	}

	return &TriggerJobProvisionChain{
		client:   client,
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

func nextServeOrNil(next jobhandler.JenkinsJobHandler, jj *v1alpha1.JenkinsJob) error {
	if next != nil {
		return next.ServeRequest(jj)
	}
	log.Info("handling of jenkins job has been finished", "name", jj.Name)
	return nil
}
