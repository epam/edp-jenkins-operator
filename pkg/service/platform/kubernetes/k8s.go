package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme     *runtime.Scheme
	CoreClient coreV1Client.CoreV1Client
}

// Init initializes K8SService
func (service *K8SService) Init(config *rest.Config, Scheme *runtime.Scheme) error {
	CoreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init core client for K8S")
	}
	service.CoreClient = *CoreClient
	service.Scheme = Scheme
	return nil
}
