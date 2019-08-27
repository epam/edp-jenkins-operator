package openshift

import (
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	"jenkins-operator/pkg/service/platform/kubernetes"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient   appsV1client.AppsV1Client
	routeClient routeV1Client.RoucteV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	err := service.K8SService.Init(config, scheme)
	if err != nil {
		return errors.Wrap(err, "Failed to init K8S platform service")
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init apps V1 client for Openshift")
	}
	service.appClient = *appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init route V1 client for Openshift")
	}
	service.routeClient = *routeClient

	return nil
}
