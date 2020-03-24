package chain

import (
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	handler2 "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins_job_handler")

func CreateDefChain(s *runtime.Scheme, c *client.Client) (handler2.JenkinsJobHandler, error) {
	pt := helper.GetPlatformTypeEnv()
	ps, err := platform.NewPlatformService(pt, s, c)
	if err != nil {
		return nil, err
	}
	cs := openshift.CreateOpenshiftClients()
	cs.Client = *c

	return PutClusterProject{
		next: PutRoleBinding{
			next: PutJenkinsPipeline{
				cs: *cs,
				ps: ps,
			},
			cs: *cs,
			ps: ps,
		},
		cs: *cs,
		ps: ps,
	}, nil

}

func nextServeOrNil(next handler2.JenkinsJobHandler, jj *v1alpha1.JenkinsJob) error {
	if next != nil {
		return next.ServeRequest(jj)
	}
	log.Info("handling of jenkins job has been finished", "name", jj.Name)
	return nil
}
