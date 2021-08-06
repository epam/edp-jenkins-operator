package platform

import (
	"errors"
	"testing"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	coreV1Api "k8s.io/api/core/v1"
)

func TestMock_All(t *testing.T) {
	m := Mock{}

	m.On("CreateSecret", v1alpha1.Jenkins{}, "test", map[string][]byte{}).Return(nil)
	if err := m.CreateSecret(v1alpha1.Jenkins{}, "test", map[string][]byte{}); err != nil {
		t.Fatal(err)
	}

	m.On("CreateJenkinsScript", "test", "test", false).Return(&v1alpha1.JenkinsScript{}, nil)
	if _, err := m.CreateJenkinsScript("test", "test", false); err != nil {
		t.Fatal(err)
	}

	m.On("CreateJenkinsScript", "test2", "test2", false).Return(nil, errors.New("fatal"))
	if _, err := m.CreateJenkinsScript("test2", "test2", false); err == nil {
		t.Fatal("no error")
	}

	m.On("CreateConfigMap", v1alpha1.Jenkins{}, "").Return(&coreV1Api.ConfigMap{}, nil)
	if _, err := m.CreateConfigMap(v1alpha1.Jenkins{}, "", map[string]string{}); err != nil {
		t.Fatal(err)
	}

	m.On("CreateConfigMap", v1alpha1.Jenkins{}, "2").Return(false, errors.New("fatal"))
	if _, err := m.CreateConfigMap(v1alpha1.Jenkins{}, "2", map[string]string{}); err == nil {
		t.Fatal("no error")
	}
}

func TestMock_CreateConfigMapWithUpdate(t *testing.T) {
	m := Mock{}

	m.On("CreateConfigMapWithUpdate", v1alpha1.Jenkins{}, "name1").Return(false, nil)
	if _, err := m.CreateConfigMapWithUpdate(v1alpha1.Jenkins{}, "name1", map[string]string{}); err != nil {
		t.Fatal(err)
	}

	m.On("CreateConfigMapWithUpdate", v1alpha1.Jenkins{}, "name2").
		Return(false, errors.New("fatal"))
	if _, err := m.CreateConfigMapWithUpdate(v1alpha1.Jenkins{}, "name2", map[string]string{}); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_AddVolumeToInitContainer(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}
	if err := m.AddVolumeToInitContainer(v1alpha1.Jenkins{}, "test", []coreV1Api.Volume{},
		[]coreV1Api.VolumeMount{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_CreateConfigMapFromFileOrDir(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if err := m.CreateConfigMapFromFileOrDir(v1alpha1.Jenkins{}, "", nil, "",
		&v1alpha1.Jenkins{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_CreateEDPComponentIfNotExist(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if err := m.CreateEDPComponentIfNotExist(v1alpha1.Jenkins{}, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetExternalEndpoint(t *testing.T) {
	m := Mock{}

	m.On("GetExternalEndpoint", "ns", "name1").Return("", "", "", nil)
	if _, _, _, err := m.GetExternalEndpoint("ns", "name1"); err != nil {
		t.Fatal(err)
	}

	m.On("GetExternalEndpoint", "ns", "name2").
		Return("", "", "", errors.New("fatal"))
	if _, _, _, err := m.GetExternalEndpoint("ns", "name2"); err == nil {
		t.Fatal("no error returned")
	}
}

func TestMock_IsDeploymentReady(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if _, err := m.IsDeploymentReady(v1alpha1.Jenkins{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetSecretData(t *testing.T) {
	m := Mock{}

	m.On("GetSecretData", "ns", "name").Return(nil, errors.New("fata"))
	if _, err := m.GetSecretData("ns", "name"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("GetSecretData", "ns", "name1").Return(map[string][]byte{"foo": []byte("bar")}, nil)
	if _, err := m.GetSecretData("ns", "name1"); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetConfigMapData(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if _, err := m.GetConfigMapData("", ""); err != nil {
		t.Fatal(err)
	}
}

func TestMock_CreateKeycloakClient(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if err := m.CreateKeycloakClient(nil); err != nil {
		t.Fatal(err)
	}
}

func TestMock_CreateStageJSON(t *testing.T) {
	defer func() {
		recover()
	}()

	m := Mock{}

	if _, err := m.CreateStageJSON(cdPipeApi.Stage{}); err != nil {
		t.Fatal(err)
	}
}

func TestMock_GetKeycloakClient(t *testing.T) {
	m := Mock{}

	m.On("GetKeycloakClient", "name1", "ns").Return(nil, errors.New("fatal"))
	if _, err := m.GetKeycloakClient("name1", "ns"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("GetKeycloakClient", "name2", "ns").Return(keycloakV1Api.KeycloakClient{}, nil)
	if _, err := m.GetKeycloakClient("name2", "ns"); err != nil {
		t.Fatal(err)
	}
}
