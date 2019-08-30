package jenkins

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	"jenkins-operator/pkg/service/platform"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins_client")

// JenkinsClient abstraction fo Jenkins client
type JenkinsClient struct {
	Instance        *v1alpha1.Jenkins
	PlatformService platform.PlatformService
	Resty           resty.Client
}

// InitNewRestClient performs initialization of Jenkins connection
func InitJenkinsClient(instance *v1alpha1.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	route, scheme, err := platformService.GetRoute(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get route for %v", instance.Name)
	}
	apiUrl := fmt.Sprintf("%v://%v", scheme, route.Spec.Host)

	if instance.Status.AdminSecretName == nil {
		log.V(1).Info("Admin secret is not created yet")
		return nil, nil
	}

	adminSecret, err := platformService.GetSecretData(instance.Namespace, *instance.Status.AdminSecretName)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get admin secret for %v", instance.Name)
	}
	jc := &JenkinsClient{
		Instance:        instance,
		PlatformService: platformService,
		Resty:           *resty.SetHostURL(apiUrl).SetBasicAuth(string(adminSecret["username"]), string(adminSecret["password"])),
	}
	return jc, nil
}
