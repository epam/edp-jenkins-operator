package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins_folder_handler")

func CreateCDPipelineFolderChain(s *runtime.Scheme, c *client.Client) (handler.JenkinsFolderHandler, error) {
	pt := helper.GetPlatformTypeEnv()
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}
	cs := openshift.CreateOpenshiftClients()
	cs.Client = *c

	return PutCDPipelineJenkinsFolder{
		cs:     *cs,
		ps:     ps,
		scheme: s,
	}, nil
}

func CreateTriggerBuildProvisionChain(s *runtime.Scheme, c *client.Client) (handler.JenkinsFolderHandler, error) {
	pt := helper.GetPlatformTypeEnv()
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}
	cs := openshift.CreateOpenshiftClients()
	cs.Client = *c

	return TriggerBuildJobProvision{
		cs: *cs,
		ps: ps,
	}, nil
}

func nextServeOrNil(next handler.JenkinsFolderHandler, jf *v1alpha1.JenkinsFolder) error {
	if next != nil {
		return next.ServeRequest(jf)
	}
	log.Info("handling of jenkins job has been finished", "name", jf.Name)
	return nil
}
