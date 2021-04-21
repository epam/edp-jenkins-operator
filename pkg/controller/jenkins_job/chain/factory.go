package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	jobhandler "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("jenkins_job_handler")

func CreateDefChain(s *runtime.Scheme, c *client.Client) (jobhandler.JenkinsJobHandler, error) {
	pt := helper.GetPlatformTypeEnv()
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}

	return PutClusterProject{
		next: PutRoleBinding{
			next: PutJenkinsPipeline{
				client: *c,
				ps:     ps,
			},
			client: *c,
			ps:     ps,
		},
		client: *c,
		ps:     ps,
	}, nil

}

func CreateTriggerJobProvisionChain(s *runtime.Scheme, c *client.Client) (jobhandler.JenkinsJobHandler, error) {
	pt := helper.GetPlatformTypeEnv()
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}

	return PutClusterProject{
		next: PutRoleBinding{
			next: TriggerJobProvision{
				client: *c,
				ps:     ps,
			},
			client: *c,
			ps:     ps,
		},
		client: *c,
		ps:     ps,
	}, nil

}

func nextServeOrNil(next jobhandler.JenkinsJobHandler, jj *v1alpha1.JenkinsJob) error {
	if next != nil {
		return next.ServeRequest(jj)
	}
	log.Info("handling of jenkins job has been finished", "name", jj.Name)
	return nil
}
