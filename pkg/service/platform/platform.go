package platform

import (
	"github.com/pkg/errors"
	"jenkins-operator/pkg/service/platform/openshift"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

// PlatformService interface
type PlatformService interface {
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
