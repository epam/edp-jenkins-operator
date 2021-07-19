package jenkins_jobbuildrun

import (
	"github.com/bndr/gojenkins"
	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type jenkinsClient interface {
	GetJobByName(jobName string) (*gojenkins.Job, error)
	BuildJob(jobName string, parameters map[string]string) (*int64, error)
	GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error)
	BuildIsRunning(build *gojenkins.Build) bool
}

type jenkinsClientFactory interface {
	MakeNewJenkinsClient(jf *v2v1alpha1.JenkinsJobBuildRun) (jenkinsClient, error)
}

type jenkinsClientBuilder struct {
	platform platform.PlatformService
	client   client.Client
}

func makeJenkinsClientBuilder(platform platform.PlatformService, client client.Client) *jenkinsClientBuilder {
	return &jenkinsClientBuilder{
		platform: platform,
		client:   client,
	}
}

func (jcb *jenkinsClientBuilder) MakeNewJenkinsClient(jf *v2v1alpha1.JenkinsJobBuildRun) (jenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(jcb.client, jf.Name, jf.Namespace, jf.Spec.OwnerName,
		jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err,
			"an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}

	cl, err := jClient.InitGoJenkinsClient(j, jcb.platform)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init go jenkins client")
	}

	return cl, nil
}
