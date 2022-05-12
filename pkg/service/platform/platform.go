package platform

import (
	"strings"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/openshift"
)

const (
	K8SPlatformType       = "kubernetes"
	OpenShiftPlatformType = "openshift"
)

// PlatformService interface
type PlatformService interface {
	CreateSecret(instance *jenkinsApi.Jenkins, name string, data map[string][]byte) error
	CreateConfigMapFromFileOrDir(instance *jenkinsApi.Jenkins, configMapName string, configMapKey *string, path string, ownerReference metav1.Object, customLabels ...map[string]string) error
	GetExternalEndpoint(namespace string, name string) (string, string, string, error)
	IsDeploymentReady(instance jenkinsApi.Jenkins) (bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	AddVolumeToInitContainer(instance *jenkinsApi.Jenkins, containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateJenkinsScript(namespace string, configMap string, forceExecute bool) (*jenkinsApi.JenkinsScript, error)
	CreateConfigMap(instance *jenkinsApi.Jenkins, name string, data map[string]string,
		labels ...map[string]string) (*coreV1Api.ConfigMap, error)
	CreateConfigMapWithUpdate(instance *jenkinsApi.Jenkins, name string, data map[string]string,
		labels ...map[string]string) (isUpdated bool, err error)
	CreateEDPComponentIfNotExist(instance jenkinsApi.Jenkins, url string, icon string) error
	CreateStageJSON(stage cdPipeApi.Stage) (string, error)
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest configs for platform")
	}

	switch strings.ToLower(platformType) {
	case OpenShiftPlatformType:
		platform := openshift.OpenshiftService{}
		err := platform.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Openshift platform")
		}
		return platform, nil
	case K8SPlatformType:
		platform := kubernetes.K8SService{}
		err := platform.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Kubernetes platform")
		}
		return platform, nil
	default:
		return nil, errors.New("Unknown platform type")
	}
}
