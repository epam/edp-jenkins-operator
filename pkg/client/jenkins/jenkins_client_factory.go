package jenkins

import (
	"github.com/bndr/gojenkins"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

type ClientInterface interface {
	GetJobByName(jobName string) (*gojenkins.Job, error)
	BuildJob(jobName string, parameters map[string]string) (*int64, error)
	GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error)
	BuildIsRunning(build *gojenkins.Build) bool
	AddRole(roleType, name, pattern string, permissions []string) error
	RemoveRoles(roleType string, roleNames []string) error
	AssignRole(roleType, roleName, subject string) error
	GetRole(roleType, roleName string) (*Role, error)
	UnAssignRole(roleType, roleName, subject string) error
}

type ClientFactory interface {
	MakeNewClient(om *metav1.ObjectMeta, ownerName *string) (ClientInterface, error)
}

type ClientBuilder struct {
	platform platform.PlatformService
	client   client.Client
}

func MakeClientBuilder(platform platform.PlatformService, client client.Client) *ClientBuilder {
	return &ClientBuilder{
		platform: platform,
		client:   client,
	}
}

func (jcb *ClientBuilder) MakeNewClient(om *metav1.ObjectMeta, ownerName *string) (ClientInterface, error) {
	j, err := plutil.GetJenkinsInstanceOwner(jcb.client, om.Name, om.Namespace, ownerName, om.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err,
			"an error has been occurred while getting owner jenkins for jenkins folder %v", om.Name)
	}

	cl, err := InitGoJenkinsClient(j, jcb.platform)
	if err != nil {
		return nil, errors.Wrap(err, "unable to init go jenkins client")
	}

	return cl, nil
}
