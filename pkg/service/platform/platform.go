package platform

import (
	routeV1Api "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	"jenkins-operator/pkg/service/platform/openshift"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

// PlatformService interface
type PlatformService interface {
	CreateServiceAccount(instance v1alpha1.Jenkins) error
	CreatePersistentVolumeClaim(instance v1alpha1.Jenkins) error
	CreateService(instance v1alpha1.Jenkins) error
	CreateSecret(instance v1alpha1.Jenkins, name string, data map[string][]byte) error
	CreateDeployConf(instance v1alpha1.Jenkins) error
	CreateExternalEndpoint(instance v1alpha1.Jenkins) error
	GetRoute(namespace string, name string) (*routeV1Api.Route, string, error)
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest config for platform")
	}

	platform := openshift.OpenshiftService{}

	err = platform.Init(restConfig, scheme)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to init for platform")
	}
	return platform, nil
}
