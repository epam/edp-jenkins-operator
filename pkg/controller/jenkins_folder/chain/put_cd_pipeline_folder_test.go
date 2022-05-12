package chain

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mocks "github.com/epam/edp-jenkins-operator/v2/mock"
	jfmock "github.com/epam/edp-jenkins-operator/v2/mock/jenkins_folder"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	URLScheme = "https"
	name      = "name"
	namespace = "namespace"
)

func nsn() types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name}
}

func ObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func TestPutCDPipelineJenkinsFolder_ServeRequest_tryToSetCDPipelineOwnerRefErr(t *testing.T) {
	jenkinsFolder := &jenkinsApi.JenkinsFolder{}

	scheme := runtime.NewScheme()
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().Build()
	platform := pmock.PlatformService{}

	p := PutCDPipelineJenkinsFolder{
		scheme: scheme,
		client: client,
		ps:     &platform,
		next:   &jenkinsFolderHandler,
	}

	err := p.ServeRequest(jenkinsFolder)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while setting owner reference"))
}

func TestPutCDPipelineJenkinsFolder_ServeRequest_initGoJenkinsClientErr(t *testing.T) {
	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta()}
	cd := &cdPipeApi.CDPipeline{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	mockClient := mocks.Client{}
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &jenkinsApi.JenkinsFolder{})
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd).Build()
	platform := pmock.PlatformService{}

	mockClient.On("Get", nsn(), &cdPipeApi.CDPipeline{}).Return(client)
	mockClient.On("Update").Return(nil)
	mockClient.On("List", &jenkinsApi.JenkinsList{}).Return(errors.New(""))

	p := PutCDPipelineJenkinsFolder{
		scheme: scheme,
		client: &mockClient,
		ps:     &platform,
		next:   &jenkinsFolderHandler,
	}

	err := p.ServeRequest(jenkinsFolder)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client"))
	mockClient.AssertExpectations(t)
}

func TestPutCDPipelineJenkinsFolder_ServeRequest_setStatusErr(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}
	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta()}
	cd := &cdPipeApi.CDPipeline{ObjectMeta: ObjectMeta()}
	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	jenkins.Status.AdminSecretName = name

	errTest := errors.New("test")

	scheme := runtime.NewScheme()
	mockClient := mocks.Client{}
	statusWriter := &mocks.StatusWriter{}
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &jenkinsApi.JenkinsFolder{}, &jenkinsApi.Jenkins{}, &jenkinsApi.JenkinsList{})
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, jenkins).Build()
	platform := pmock.PlatformService{}

	mockClient.On("Get", nsn(), &cdPipeApi.CDPipeline{}).Return(client)
	mockClient.On("Update").Return(nil).Once()
	mockClient.On("List", &jenkinsApi.JenkinsList{}).Return(client)
	mockClient.On("Get", nsn(), &jenkinsApi.Jenkins{}).Return(client)
	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)
	mockClient.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	mockClient.On("Update").Return(errTest)

	innerJob := gojenkins.InnerJob{Name: name}
	Raw := gojenkins.ExecutorResponse{Jobs: []gojenkins.InnerJob{innerJob}}
	marshal, err := json.Marshal(Raw)
	assert.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewBytesResponder(http.StatusOK, marshal))

	p := PutCDPipelineJenkinsFolder{
		scheme: scheme,
		client: &mockClient,
		ps:     &platform,
		next:   &jenkinsFolderHandler,
	}
	err = p.ServeRequest(jenkinsFolder)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while updating"))
	mockClient.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
	platform.AssertExpectations(t)
}

func TestPutCDPipelineJenkinsFolder_ServeRequest(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}
	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta()}
	cd := &cdPipeApi.CDPipeline{ObjectMeta: ObjectMeta()}

	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	jenkins.Status.AdminSecretName = name

	errTest := errors.New("test")

	scheme := runtime.NewScheme()
	mockClient := mocks.Client{}
	statusWriter := &mocks.StatusWriter{}
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &jenkinsApi.JenkinsFolder{}, &jenkinsApi.Jenkins{}, &jenkinsApi.JenkinsList{})
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cd, jenkins).Build()
	platform := pmock.PlatformService{}

	mockClient.On("Get", nsn(), &cdPipeApi.CDPipeline{}).Return(client)
	mockClient.On("Update").Return(nil)
	mockClient.On("List", &jenkinsApi.JenkinsList{}).Return(client)
	mockClient.On("Get", nsn(), &jenkinsApi.Jenkins{}).Return(client)
	platform.On("GetExternalEndpoint", namespace, name).Return("a", URLScheme, "b", nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)
	mockClient.On("Status").Return(statusWriter)
	statusWriter.On("Update").Return(errTest)
	jenkinsFolderHandler.On("ServeRequest", jenkinsFolder).Return(nil)

	innerJob := gojenkins.InnerJob{Name: name}
	Raw := gojenkins.ExecutorResponse{Jobs: []gojenkins.InnerJob{innerJob}}
	marshal, err := json.Marshal(Raw)
	assert.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, "https://ab/api/json", httpmock.NewBytesResponder(http.StatusOK, marshal))

	p := PutCDPipelineJenkinsFolder{
		scheme: scheme,
		client: &mockClient,
		ps:     &platform,
		next:   &jenkinsFolderHandler,
	}
	err = p.ServeRequest(jenkinsFolder)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	statusWriter.AssertExpectations(t)
	platform.AssertExpectations(t)
}
