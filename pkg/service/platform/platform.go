package platform

import (
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/openshift"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	appsV1Api "github.com/openshift/api/apps/v1"
	authV1Api "github.com/openshift/api/authorization/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PlatformService interface
type PlatformService interface {
	CreateServiceAccount(instance v1alpha1.Jenkins) error
	CreatePersistentVolumeClaim(instance v1alpha1.Jenkins) error
	CreateService(instance v1alpha1.Jenkins) error
	CreateSecret(instance v1alpha1.Jenkins, name string, data map[string][]byte) error
	CreateDeployConf(instance v1alpha1.Jenkins) error
	CreateExternalEndpoint(instance v1alpha1.Jenkins) error
	CreateConfigMapFromFileOrDir(instance v1alpha1.Jenkins, configMapName string, configMapKey *string, path string, ownerReference metav1.Object, customLabels ...map[string]string) error
	CreateConfigMapFromData(instance v1alpha1.Jenkins, configMapName string, configMapData map[string]string, labels map[string]string, ownerReference metav1.Object) error
	CreateRole(ac v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error
	CreateClusterRole(ac v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error
	GetRoute(namespace string, name string) (*routeV1Api.Route, string, error)
	GetDeploymentConfig(instance v1alpha1.Jenkins) (*appsV1Api.DeploymentConfig, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateUserRoleBinding(instance v1alpha1.Jenkins, name string, binding string, kind string) error
	CreateUserClusterRoleBinding(instance v1alpha1.Jenkins, name string, binding string) error
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	AddVolumeToInitContainer(instance v1alpha1.Jenkins, dc *appsV1Api.DeploymentConfig, containerName string, vol []coreV1Api.Volume,
		volMount []coreV1Api.VolumeMount) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest configs for platform")
	}

	platform := openshift.OpenshiftService{}

	err = platform.Init(restConfig, scheme, k8sClient)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to init for platform")
	}
	return platform, nil
}
