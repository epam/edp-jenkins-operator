package platform

import (
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/openshift"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	authV1Api "k8s.io/api/rbac/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// PlatformService interface
type PlatformService interface {
	CreateServiceAccount(instance v1alpha1.Jenkins) error
	CreatePersistentVolumeClaim(instance v1alpha1.Jenkins) error
	CreateService(instance v1alpha1.Jenkins) error
	CreateSecret(instance v1alpha1.Jenkins, name string, data map[string][]byte) error
	CreateDeployment(instance v1alpha1.Jenkins) error
	CreateExternalEndpoint(instance v1alpha1.Jenkins) error
	CreateConfigMapFromFileOrDir(instance v1alpha1.Jenkins, configMapName string, configMapKey *string, path string, ownerReference metav1.Object, customLabels ...map[string]string) error
	CreateRole(ac v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error
	CreateClusterRolePolicyRules() []authV1Api.PolicyRule
	CreateClusterRole(ac v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error
	GetExternalEndpoint(namespace string, name string) (string, string, string, error)
	IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error)
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	CreateUserRoleBinding(instance v1alpha1.Jenkins, name string, binding string, kind string) error
	CreateUserClusterRoleBinding(instance v1alpha1.Jenkins, name string, binding string) error
	GetConfigMapData(namespace string, name string) (map[string]string, error)
	AddVolumeToInitContainer(instance v1alpha1.Jenkins, containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateJenkinsScript(namespace string, configMap string) (*v1alpha1.JenkinsScript, error)
	CreateConfigMap(instance v1alpha1.Jenkins, configMapName string, configMapData map[string]string, labels ...map[string]string) error
	CreateEDPComponentIfNotExist(instance v1alpha1.Jenkins, url string, icon string) error
	CreateProject(name string) error
	CreateRoleBinding(edpName, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error
	CreateStageJSON(cr edpv1alpha1.Stage) (string, error)
	DeleteProject(name string) error
	GetRoleBinding(roleBindingName, namespace string) (*rbacV1.RoleBinding, error)
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest configs for platform")
	}

	switch strings.ToLower(platformType) {
	case "openshift":
		platform := openshift.OpenshiftService{}
		err := platform.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Openshift platform")
		}
		return platform, nil
	case "kubernetes":
		platform := kubernetes.K8SService{}
		err := platform.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Kubernetes platform")
		}
		return platform, nil
	default:
		return nil, errors.Wrap(err, "Unknown platform type")
	}
}
