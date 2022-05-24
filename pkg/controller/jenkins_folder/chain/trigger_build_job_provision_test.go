package chain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jfmock "github.com/epam/edp-jenkins-operator/v2/mock/jenkins_folder"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestTriggerBuildJobProvision_ServeRequest_setStatusErr(t *testing.T) {
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().Build()
	platform := pmock.PlatformService{}

	jf := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{}}}
	jf.Spec.Job.Name = name

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}
	err := tr.ServeRequest(jf)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while updating"))
}

func TestTriggerBuildJobProvision_ServeRequest_triggerBuildJobProvisionErr(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{})

	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{}}}
	jenkinsFolder.Spec.Job.Name = name

	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsFolder).Build()
	platform := pmock.PlatformService{}

	trigger := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}
	err := trigger.ServeRequest(jenkinsFolder)
	fmt.Println(err.Error())
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client"))
}

func TestTriggerBuildJobProvision_ServeRequest_setStatusErr2(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	resp := gojenkins.BuildResponse{Result: "SUCCESS"}
	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkins).Build()
	platform := pmock.PlatformService{}

	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{}}}
	jenkinsFolder.Spec.Job.Name = name
	ownerReference := v1.OwnerReference{Kind: "Jenkins", Name: name}
	jenkinsFolder.ObjectMeta.OwnerReferences = []v1.OwnerReference{ownerReference}

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)
	httpmock.RegisterResponder(http.MethodGet, "https:////api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https:////job/name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https:////job/name/0/api/json?depth=1", httpmock.NewBytesResponder(http.StatusOK, raw))

	trigger := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}
	err = trigger.ServeRequest(jenkinsFolder)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while updating"))
	platform.AssertExpectations(t)
}

func TestTriggerBuildJobProvision_ServeRequest_UnmarshallErr(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{}, &jenkinsApi.JenkinsFolder{})
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{}}}
	jenkinsFolder.Spec.Job.Name = name
	ownerReference := v1.OwnerReference{Kind: "Jenkins", Name: name}
	jenkinsFolder.ObjectMeta.OwnerReferences = []v1.OwnerReference{ownerReference}

	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkins, jenkinsFolder).Build()
	platform := pmock.PlatformService{}

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/name/0/api/json?depth=1", httpmock.NewStringResponder(http.StatusOK, ""))

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}
	err := tr.ServeRequest(jenkinsFolder)
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	platform.AssertExpectations(t)
}

func TestTriggerBuildJobProvision_ServeRequest(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{}, &jenkinsApi.JenkinsFolder{})
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	jenkinsFolder := &jenkinsApi.JenkinsFolder{ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{}}}
	jenkinsFolder.Spec.Job.Name = name

	resp := gojenkins.BuildResponse{Result: "SUCCESS"}
	raw, err := json.Marshal(resp)
	assert.NoError(t, err)

	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkins, jenkinsFolder).Build()
	platform := pmock.PlatformService{}

	ownerReference := v1.OwnerReference{Kind: "Jenkins", Name: name}
	jenkinsFolder.ObjectMeta.OwnerReferences = []v1.OwnerReference{ownerReference}

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/name/0/api/json?depth=1", httpmock.NewBytesResponder(http.StatusOK, raw))
	jenkinsFolderHandler.On("ServeRequest", jenkinsFolder).Return(nil)

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}
	err = tr.ServeRequest(jenkinsFolder)
	assert.NoError(t, err)
	platform.AssertExpectations(t)
	jenkinsFolderHandler.AssertExpectations(t)
}

func TestTriggerBuildJobProvision_ServeRequest_SkipBuild(t *testing.T) {
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().Build()
	platform := pmock.PlatformService{}

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}

	err := tr.ServeRequest(&jenkinsApi.JenkinsFolder{})

	assert.Error(t, err)
}
