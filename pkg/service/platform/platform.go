package platform

import (
	"errors"
	"fmt"
	"strings"

	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/openshift"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
)

const (
	K8SPlatformType       = "kubernetes"
	OpenShiftPlatformType = "openshift"
)

// PlatformService interface.
type PlatformService interface {
	CreateSecret(instance *jenkinsApi.Jenkins, name string, data map[string][]byte) error
	CreateConfigMapFromFileOrDir(instance *jenkinsApi.Jenkins, configMapName string, configMapKey *string, path string, ownerReference metav1.Object, customLabels ...map[string]string) error
	GetExternalEndpoint(namespace, name string) (string, string, string, error)
	IsDeploymentReady(instance *jenkinsApi.Jenkins) (bool, error)
	GetSecretData(namespace, name string) (map[string][]byte, error)
	GetConfigMapData(namespace, name string) (map[string]string, error)
	AddVolumeToInitContainer(instance *jenkinsApi.Jenkins, containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateJenkinsScript(namespace string, configMap string, forceExecute bool) (*jenkinsApi.JenkinsScript, error)
	CreateConfigMap(instance *jenkinsApi.Jenkins, name string, data map[string]string,
		labels ...map[string]string) (*coreV1Api.ConfigMap, error)
	CreateConfigMapWithUpdate(instance *jenkinsApi.Jenkins, name string, data map[string]string,
		labels ...map[string]string) (isUpdated bool, err error)
	CreateEDPComponentIfNotExist(instance *jenkinsApi.Jenkins, url, icon string) error
	CreateStageJSON(stage *cdPipeApi.Stage) (string, error)
}

// NewPlatformService returns platform service interface implementation.
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest configs for platform: %w", err)
	}

	switch strings.ToLower(platformType) {
	case OpenShiftPlatformType:
		platform := openshift.OpenshiftService{}

		if err := platform.Init(restConfig, scheme, k8sClient); err != nil {
			return nil, fmt.Errorf("failed to init Openshift platform: %w", err)
		}

		return &platform, nil
	case K8SPlatformType:
		platform := kubernetes.K8SService{}

		if err := platform.Init(restConfig, scheme, k8sClient); err != nil {
			return nil, fmt.Errorf(" failed to init Kubernetes platform: %w", err)
		}

		return &platform, nil
	default:
		return nil, errors.New("received unknown platform type")
	}
}
