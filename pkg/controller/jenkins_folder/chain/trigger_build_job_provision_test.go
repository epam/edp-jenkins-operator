package chain

import (
	"encoding/json"
	"net/http"
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

	jf := &jenkinsApi.JenkinsFolder{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{},
		},
	}
	jf.Spec.Job.Name = name

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}

	err := tr.ServeRequest(jf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update")
}

func TestTriggerBuildJobProvision_ServeRequest_triggerBuildJobProvisionErr(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{})

	jenkinsFolder := &jenkinsApi.JenkinsFolder{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{},
		},
	}
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gojenkins client")
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

	jenkinsFolder := &jenkinsApi.JenkinsFolder{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{},
		},
	}
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
	assert.Contains(t, err.Error(), "failed to update")
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

	jenkinsFolder := &jenkinsApi.JenkinsFolder{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{},
		},
	}
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

	assert.Contains(t, tr.ServeRequest(jenkinsFolder).Error(), "unexpected end of JSON input")
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

	data := map[string]string{"str1": "str2"}

	raw, err := json.Marshal(data)
	assert.NoError(t, err)

	jenkinsFolder := &jenkinsApi.JenkinsFolder{
		ObjectMeta: ObjectMeta(),
		Spec: jenkinsApi.JenkinsFolderSpec{
			Job: &jenkinsApi.Job{
				Name:   name,
				Config: string(raw),
			},
		},
	}

	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}

	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkins, jenkinsFolder).Build()
	platform := pmock.PlatformService{}

	ownerReference := v1.OwnerReference{Kind: "Jenkins", Name: name}
	jenkinsFolder.ObjectMeta.OwnerReferences = []v1.OwnerReference{ownerReference}

	// look at taskResponse struct from goJenkins queue.go
	taskResponseRaw := []byte("{\"executable\":{\"number\":1,\"url\":\"\"}}")

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://queue/item/0/api/json", httpmock.NewBytesResponder(http.StatusOK, taskResponseRaw))
	jenkinsFolderHandler.On("ServeRequest", jenkinsFolder).Return(nil)

	tr := TriggerBuildJobProvision{
		next:   &jenkinsFolderHandler,
		client: client,
		ps:     &platform,
	}

	assert.NoError(t, tr.ServeRequest(jenkinsFolder))
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

	assert.Error(t, tr.ServeRequest(&jenkinsApi.JenkinsFolder{}))
}
