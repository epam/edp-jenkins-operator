package jenkins

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1Api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritSpec "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	keycloakControllerHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"

	mocks "github.com/epam/edp-jenkins-operator/v2/mock"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
)

const (
	name      = "name"
	namespace = "namespace"
	URLScheme = "https"
	urlName   = "example"
	domain    = ".com"
)

func ObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func CreateKeycloakClient() *keycloakV1Api.KeycloakClient {
	return &keycloakV1Api.KeycloakClient{
		TypeMeta:   v1.TypeMeta{},
		ObjectMeta: ObjectMeta(),
		Spec: keycloakV1Api.KeycloakClientSpec{
			ClientId: name,
			Public:   true,
			WebUrl:   "https://example.com",
			RealmRoles: &[]keycloakV1Api.RealmRole{
				{
					Name:      "jenkins-administrators",
					Composite: "administrator",
				},
				{
					Name:      "jenkins-users",
					Composite: "developer",
				},
			},
		},
	}
}

func TestJenkinsServiceImpl_createTemplateScript(t *testing.T) {
	ji := &jenkinsApi.Jenkins{}
	platformMock := pmock.PlatformService{}
	jenkinsScriptData := platformHelper.JenkinsScriptData{}
	data := map[string]string{"context": "lol"}

	fp, err := os.Create("/tmp/temp.tpl")
	require.NoError(t, err)

	_, err = fp.WriteString("lol")
	require.NoError(t, err)

	require.NoError(t, fp.Close())

	defer func() {
		require.NoError(t, os.Remove("/tmp/temp.tpl"))
	}()

	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp", data).
		Return(false, nil)
	platformMock.On("CreateJenkinsScript", "", "-temp", false).
		Return(&jenkinsApi.JenkinsScript{}, nil)

	require.NoError(t, createTemplateScript(
		"/tmp",
		"temp.tpl",
		&platformMock,
		&jenkinsScriptData,
		ji,
	))
}

func TestJenkinsServiceImpl_createTemplateScript_Failure(t *testing.T) {
	ji := &jenkinsApi.Jenkins{
		Spec: jenkinsApi.JenkinsSpec{},
	}
	data := map[string]string{"context": "lol"}
	platformMock := pmock.PlatformService{}
	jenkinsScriptData := platformHelper.JenkinsScriptData{}

	err := createTemplateScript(
		"/tmp", "temp123.tpl",
		&platformMock,
		&jenkinsScriptData,
		ji,
	)
	require.Error(t, err)

	require.Contains(t, err.Error(), "ailed to find template file in specified pathToTemplate")

	fp, err := os.Create("/tmp/temp.tpl")
	require.NoError(t, err)

	_, err = fp.WriteString("lol")
	require.NoError(t, err)

	err = fp.Close()
	require.NoError(t, err)

	defer func() {
		removeErr := os.Remove("/tmp/temp.tpl")
		require.NoError(t, removeErr)
	}()

	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp", data).
		Return(false, fmt.Errorf("CreateConfigMap fatal")).
		Once()

	err = createTemplateScript("/tmp", "temp.tpl", &platformMock, &jenkinsScriptData, ji)
	require.Error(t, err)

	require.Contains(t, err.Error(), "CreateConfigMap fatal")

	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp", data).
		Return(false, nil)
	platformMock.On("CreateJenkinsScript", "", "-temp", false).
		Return(nil, fmt.Errorf("CreateJenkinsScript fatal"))

	err = createTemplateScript("/tmp", "temp.tpl", &platformMock, &jenkinsScriptData, ji)
	require.Error(t, err)

	require.Contains(t, err.Error(), "CreateJenkinsScript fatal")

	platformMock.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Integration_GetExternalEndpointErr(t *testing.T) {
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{Enabled: true},
		},
	}
	platform := pmock.PlatformService{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	platform.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", fmt.Errorf("test"))

	_, _, err := impl.Integration(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get route from cluster")
}

func TestJenkinsServiceImpl_CreateEDPComponent(t *testing.T) {
	impl := JenkinsServiceImpl{}
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec:       jenkinsApi.JenkinsSpec{ExternalURL: "https://google.com"},
	}
	err := impl.createEDPComponent(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jenkins.svg")
}

func TestJenkinsServiceImpl_Integration_mountGerritCredentialsErr(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &gerritApi.GerritList{}, &gerritApi.Gerrit{})

	gerritSpecName := fmt.Sprintf("%v/%v", gerritSpec.EdpAnnotationsPrefix, gerritSpec.EdpCiUSerSshKeySuffix)
	gerrit := &gerritApi.Gerrit{
		ObjectMeta: v1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{gerritSpecName: name},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gerrit).Build()

	volMount := []coreV1Api.VolumeMount{
		{
			Name:      name,
			MountPath: sshKeyDefaultMountPath,
			ReadOnly:  true,
		},
	}

	mode := int32(400)
	vol := []coreV1Api.Volume{
		{
			Name: name,
			VolumeSource: coreV1Api.VolumeSource{
				Secret: &coreV1Api.SecretVolumeSource{
					SecretName:  name,
					DefaultMode: &mode,
					Items: []coreV1Api.KeyToPath{
						{
							Key:  "id_rsa",
							Path: "id_rsa",
							Mode: &mode,
						},
					},
				},
			},
		},
	}

	instance := jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	platform := pmock.PlatformService{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
		k8sClient:       client,
		k8sScheme:       scheme,
	}

	platform.On("AddVolumeToInitContainer", &instance, "grant-permissions", vol, volMount).
		Return(fmt.Errorf("test"))

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to mount Gerrit credentials")
}

func TestJenkinsServiceImpl_Integration(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &gerritApi.GerritList{}, &gerritApi.Gerrit{})

	gerritSpecName := fmt.Sprintf("%v/%v", gerritSpec.EdpAnnotationsPrefix, gerritSpec.EdpCiUSerSshKeySuffix)
	gerrit := &gerritApi.Gerrit{
		ObjectMeta: v1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{gerritSpecName: name},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gerrit).Build()

	volMount := []coreV1Api.VolumeMount{
		{
			Name:      name,
			MountPath: sshKeyDefaultMountPath,
			ReadOnly:  true,
		},
	}

	mode := int32(400)
	vol := []coreV1Api.Volume{
		{
			Name: name,
			VolumeSource: coreV1Api.VolumeSource{
				Secret: &coreV1Api.SecretVolumeSource{
					SecretName:  name,
					DefaultMode: &mode,
					Items: []coreV1Api.KeyToPath{
						{
							Key:  "id_rsa",
							Path: "id_rsa",
							Mode: &mode,
						},
					},
				},
			},
		},
	}

	instance := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	platform := pmock.PlatformService{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
		k8sClient:       client,
		k8sScheme:       scheme,
	}

	platform.On("AddVolumeToInitContainer", instance, "grant-permissions", vol, volMount).
		Return(nil)

	_, ok, err := impl.Integration(instance)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestJenkinsServiceImpl_Integration_CreateKeycloakClientErr(t *testing.T) {
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
	}
	keycloakClient := CreateKeycloakClient()

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", keycloakClient).Return(fmt.Errorf("test"))

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Keycloak Client data")
}

func TestJenkinsServiceImpl_Integration_GetKeycloakClientErr(t *testing.T) {
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
	}
	keycloakClient := CreateKeycloakClient()

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", keycloakClient).Return(nil)
	platform.On("GetKeycloakClient", name, namespace).Return(*keycloakClient, fmt.Errorf("test"))

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Keycloak Client CR")
}

func TestJenkinsServiceImpl_Integration_GetOwnerKeycloakRealmErr(t *testing.T) {
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	keycloakHelper := keycloakControllerHelper.Helper{}
	impl := JenkinsServiceImpl{
		platformService: &platform,
		keycloakHelper:  &keycloakHelper,
	}

	keycloakClient := CreateKeycloakClient()

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", keycloakClient).Return(nil)
	platform.On("GetKeycloakClient", name, namespace).Return(*keycloakClient, nil)

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Keycloak Realm for")
}

func TestJenkinsServiceImpl_Integration_GetOwnerKeycloakErr(t *testing.T) {
	keycloakRealm := &keycloakV1Api.KeycloakRealm{
		ObjectMeta: ObjectMeta(),
	}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakRealm{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(keycloakRealm).Build()
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	keycloakHelper := keycloakControllerHelper.MakeHelper(client, scheme, logr.Discard())
	impl := JenkinsServiceImpl{
		platformService: &platform,
		keycloakHelper:  keycloakHelper,
	}
	ownerReference := v1.OwnerReference{Kind: "KeycloakRealm", Name: name}
	keycloakClient := *CreateKeycloakClient()

	keycloakClient2 := keycloakClient
	keycloakClient2.OwnerReferences = []v1.OwnerReference{ownerReference}

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", &keycloakClient).Return(nil)
	platform.On("GetKeycloakClient", name, namespace).Return(keycloakClient2, nil)

	_, _, err := impl.Integration(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get owner for ")
}

func TestJenkinsServiceImpl_Integration_ParseTemplateErr(t *testing.T) {
	ownerReference := v1.OwnerReference{Kind: "Keycloak", Name: name}

	keycloakRealm := &keycloakV1Api.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []v1.OwnerReference{ownerReference},
		},
	}
	keycloak := &keycloakV1Api.Keycloak{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakRealm{}, &keycloakV1Api.Keycloak{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(keycloakRealm, keycloak).Build()
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	keycloakHelper := keycloakControllerHelper.MakeHelper(client, scheme, logr.Discard())
	impl := JenkinsServiceImpl{
		platformService: &platform,
		keycloakHelper:  keycloakHelper,
	}
	ownerReferenceRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}

	keycloakClient := *CreateKeycloakClient()

	keycloakClient2 := keycloakClient
	keycloakClient2.OwnerReferences = []v1.OwnerReference{ownerReferenceRealm}

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", &keycloakClient).Return(nil)
	platform.On("GetKeycloakClient", name, namespace).Return(keycloakClient2, nil)

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find template file in specified pathToTemplate")
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Integration_GetSecretDataErr(t *testing.T) {
	ownerReference := v1.OwnerReference{
		Kind: "Keycloak",
		Name: name,
	}

	keycloakRealm := &keycloakV1Api.KeycloakRealm{
		ObjectMeta: v1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []v1.OwnerReference{ownerReference},
		},
	}
	keycloak := &keycloakV1Api.Keycloak{ObjectMeta: ObjectMeta()}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakRealm{}, &keycloakV1Api.Keycloak{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(keycloakRealm, keycloak).Build()
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsSpec{
			KeycloakSpec: jenkinsApi.KeycloakSpec{
				Enabled:   true,
				IsPrivate: true,
			},
		},
	}
	platform := pmock.PlatformService{}
	keycloakHelper := keycloakControllerHelper.MakeHelper(client, scheme, logr.Discard())
	impl := JenkinsServiceImpl{
		platformService: &platform,
		keycloakHelper:  keycloakHelper,
	}
	ownerReferenceRealm := v1.OwnerReference{
		Kind: "KeycloakRealm",
		Name: name,
	}

	keycloakClient := *CreateKeycloakClient()
	keycloakClient.Spec.Public = false

	keycloakClient2 := keycloakClient
	keycloakClient2.OwnerReferences = []v1.OwnerReference{ownerReferenceRealm}

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("CreateKeycloakClient", &keycloakClient).Return(nil)
	platform.On("GetKeycloakClient", name, namespace).Return(keycloakClient2, nil)
	platform.On("GetSecretData", namespace, "").Return(nil, fmt.Errorf("test"))

	_, _, err := impl.Integration(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get keycloak client secret data")
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_ExposeConfiguration_InitJenkinsClientErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	platform.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", fmt.Errorf("test"))

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to init Jenkins REST client")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_ExposeConfiguration_NilClientErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	platform.On("GetExternalEndpoint", namespace, name).
		Return(urlName, URLScheme, domain, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jenkins returns nil client")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_ExposeConfiguration_GetSlavesErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.ExposeConfiguration(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Jenkins slave list")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Configure_InitJenkinsClientErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	platform.On("GetSecretData", namespace, "name-admin-token").Return(nil, nil)
	platform.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", fmt.Errorf("test"))

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to init Jenkins REST client")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Configure_NilClientErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	platform.On("GetSecretData", namespace, "name-admin-token").Return(nil, nil)
	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jenkins returns nil client")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Configure_GetSecretDataErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	adminTokenSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
	platform.On("GetSecretData", namespace, adminTokenSecretName).Return(secretData, fmt.Errorf("test"))

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get admin token secret for")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Configure_GetAdminTokenErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	platform.On("GetExternalEndpoint", namespace, name).Return(urlName, URLScheme, domain, nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)

	adminTokenSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
	platform.On("GetSecretData", namespace, adminTokenSecretName).Return(nil, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get token from admin user")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_Configure_ReadDirErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	adminTokenSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
	platform.On("GetSecretData", namespace, adminTokenSecretName).Return(secretData, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	configuration, b, err := impl.Configure(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read scriptFiles from dir")
	assert.False(t, b)
	assert.Equal(t, instance, configuration)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_IsDeploymentReady(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	platform.On("IsDeploymentReady", &instance).Return(true, nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}
	ready, err := impl.IsDeploymentReady(&instance)
	assert.NoError(t, err)
	assert.True(t, ready)
	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_CreateAdminPassword_CreateSecretErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	platform.On("CreateSecret", &instance, "name-admin-password", mock.AnythingOfType("map[string][]uint8")).
		Return(fmt.Errorf("test"))

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	err := impl.CreateAdminPassword(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create admin password secret")

	platform.AssertExpectations(t)
}

func TestJenkinsServiceImpl_setAdminSecretInStatusErr(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	statusWriter := &mocks.StatusWriter{}
	client := mocks.Client{}

	platform.On("CreateSecret", &instance, "name-admin-password", mock.AnythingOfType("map[string][]uint8")).
		Return(nil)
	client.On("Update").Return(fmt.Errorf("test"))
	client.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(fmt.Errorf("test"))

	impl := JenkinsServiceImpl{
		platformService: &platform,
		k8sClient:       &client,
	}

	err := impl.CreateAdminPassword(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set admin secret name in status")

	platform.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestJenkinsServiceImpl_CreateAdminPassword(t *testing.T) {
	platform := pmock.PlatformService{}
	instance := jenkinsApi.Jenkins{
		ObjectMeta: ObjectMeta(),
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}

	platform.On("CreateSecret", &instance, "name-admin-password", mock.AnythingOfType("map[string][]uint8")).
		Return(nil)

	impl := JenkinsServiceImpl{
		platformService: &platform,
	}

	err := impl.CreateAdminPassword(&instance)
	assert.NoError(t, err)

	platform.AssertExpectations(t)
}

func Test_setAnnotation(t *testing.T) {
	key := "key"
	value := "val"
	instance := jenkinsApi.Jenkins{}
	setAnnotation(&instance, key, value)
	assert.Equal(t, value, instance.Annotations[key])
}
