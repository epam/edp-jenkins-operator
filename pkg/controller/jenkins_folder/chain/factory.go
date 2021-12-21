package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("jenkins-folder-chain")

func CreateCDPipelineFolderChain(s *runtime.Scheme, c *client.Client) (handler.JenkinsFolderHandler, error) {
	pt, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return nil, err
	}
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}

	return PutCDPipelineJenkinsFolder{
		client: *c,
		ps:     ps,
		scheme: s,
	}, nil
}

func CreateTriggerBuildProvisionChain(s *runtime.Scheme, c *client.Client) (handler.JenkinsFolderHandler, error) {
	pt, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return nil, err
	}
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}

	return TriggerBuildJobProvision{
		client: *c,
		ps:     ps,
	}, nil
}

func nextServeOrNil(next handler.JenkinsFolderHandler, jf *v1alpha1.JenkinsFolder) error {
	if next != nil {
		return next.ServeRequest(jf)
	}
	log.Info("handling of jenkins job has been finished", "name", jf.Name)
	return nil
}
