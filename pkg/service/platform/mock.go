package platform

import (
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/stretchr/testify/mock"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) CreateSecret(instance v1alpha1.Jenkins, name string, data map[string][]byte) error {
	return m.Called(instance, name, data).Error(0)
}

func (m *Mock) CreateConfigMapFromFileOrDir(instance v1alpha1.Jenkins, configMapName string, configMapKey *string,
	path string, ownerReference metav1.Object, customLabels ...map[string]string) error {
	panic("not implemented")
}

func (m *Mock) GetExternalEndpoint(namespace string, name string) (string, string, string, error) {
	called := m.Called(namespace, name)
	if err := called.Error(3); err != nil {
		return "", "", "", err
	}

	return called.String(0), called.String(1), called.String(2), nil
}

func (m *Mock) IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error) {
	panic("not implemented")
}

func (m *Mock) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	called := m.Called(namespace, name)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(map[string][]byte), nil
}

func (m *Mock) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	panic("not implemented")
}

func (m *Mock) AddVolumeToInitContainer(instance v1alpha1.Jenkins, containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error {
	panic("not implemented")
}

func (m *Mock) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	return m.Called(kc).Error(0)
}

func (m *Mock) GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error) {
	called := m.Called(name, namespace)
	if err := called.Error(1); err != nil {
		return keycloakV1Api.KeycloakClient{}, err
	}

	return called.Get(0).(keycloakV1Api.KeycloakClient), nil
}

func (m *Mock) CreateJenkinsScript(namespace string, configMap string, forceExecute bool) (*v1alpha1.JenkinsScript, error) {
	called := m.Called(namespace, configMap, forceExecute)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*v1alpha1.JenkinsScript), nil
}

func (m *Mock) CreateConfigMap(instance v1alpha1.Jenkins, name string, data map[string]string,
	labels ...map[string]string) (*coreV1Api.ConfigMap, error) {
	called := m.Called(instance, name)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*coreV1Api.ConfigMap), nil
}

func (m *Mock) CreateConfigMapWithUpdate(instance v1alpha1.Jenkins, name string, data map[string]string,
	labels ...map[string]string) (isUpdated bool, err error) {
	called := m.Called(instance, name)
	if err := called.Error(1); err != nil {
		return false, err
	}

	return called.Bool(0), nil
}

func (m *Mock) CreateEDPComponentIfNotExist(instance v1alpha1.Jenkins, url string, icon string) error {
	panic("not implemented")
}

func (m *Mock) CreateStageJSON(stage cdPipeApi.Stage) (string, error) {
	panic("not implemented")
}
