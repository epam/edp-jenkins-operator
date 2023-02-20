package jenkins

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/resty.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	name      = "name"
	namespace = "namespace"
	str       = `{"crumb": "file"}`
)

func createMockClient() (*gojenkins.Jenkins, error) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder(
		http.MethodGet,
		"https://api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))

	return gojenkins.CreateJenkins(http.DefaultClient, "https://").Init()
}

func CreateMockResty() *resty.Client {
	restyClient := resty.New()

	httpmock.DeactivateAndReset()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	return restyClient
}

func TestInitJenkinsClient_GetExternalEndpointErr(t *testing.T) {
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	platformService.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", fmt.Errorf("test"))

	_, err := InitJenkinsClient(instance, &platformService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to get route for name")

	platformService.AssertExpectations(t)
}

func TestInitJenkinsClient_EmptySecretName(t *testing.T) {
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance.Status.AdminSecretName = ""

	platformService.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", nil)

	_, err := InitJenkinsClient(instance, &platformService)
	assert.NoError(t, err)
	platformService.AssertExpectations(t)
}

func TestInitJenkinsClient_GetSecretDataErr(t *testing.T) {
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance.Status.AdminSecretName = name

	platformService.On("GetExternalEndpoint", namespace, name).
		Return("", "", "", nil)
	platformService.On("GetSecretData", namespace, name).
		Return(nil, fmt.Errorf("test"))

	_, err := InitJenkinsClient(instance, &platformService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get admin secret for")
	platformService.AssertExpectations(t)
}

func TestJenkinsClient_GetCrumb_GetErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	crumb, err := jc.GetCrumb()
	assert.Error(t, err)
	assert.Empty(t, crumb)
}

func TestJenkinsClient_GetCrumb_NoFoundCode(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	crumb, err := jc.GetCrumb()
	assert.NoError(t, err)
	assert.Empty(t, crumb)
}

func TestJenkinsClient_GetCrumb_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	crumb, err := jc.GetCrumb()
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
	assert.Empty(t, crumb)
}

func TestJenkinsClient_GetCrumb(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	jc := JenkinsClient{
		resty: restyClient,
	}

	crumb, err := jc.GetCrumb()
	assert.NoError(t, err)
	assert.Equal(t, "file", crumb)
}

func TestJenkinsClient_RunScript_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	script := "test"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusBadGateway, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	assert.Error(t, jc.RunScript(script))
}

func TestJenkinsClient_RunScript_PostErr(t *testing.T) {
	restyClient := CreateMockResty()
	script := "test"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	jc := JenkinsClient{
		resty: restyClient,
	}

	err := jc.RunScript(script)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform request to Jenkins script API")
}

func TestJenkinsClient_RunScript_NotFoundCode(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"/scriptText",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	err := jc.RunScript("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run script in Jenkin")
}

func TestJenkinsClient_RunScript(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"/scriptText",
		httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	assert.NoError(t, jc.RunScript("test"))
}

func TestJenkinsClient_GetSlaves_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetSlaves()
	assert.Error(t, err)
}

func TestJenkinsClient_GetSlavesErr(t *testing.T) {
	restyClient := CreateMockResty()

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetSlaves()
	assert.Error(t, err)
}

func TestJenkinsClient_CreateUser_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	instance := &jenkinsApi.JenkinsServiceAccount{}
	jc := JenkinsClient{
		resty: restyClient,
	}

	assert.Error(t, jc.CreateUser(instance))
}

func TestJenkinsClient_CreateUser_GetSecretDataErr(t *testing.T) {
	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).
		Return(nil, fmt.Errorf("test"))

	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}

	err := jc.CreateUser(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get info from secret name")
}

func TestJenkinsClient_CreateUser_NewJenkinsUserErr(t *testing.T) {
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).
		Return(secretData, nil)

	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}

	assert.Error(t, jc.CreateUser(instance))
}

func TestJenkinsClient_CreateUser_PostErr(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).
		Return(secretData, nil)

	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}

	err := jc.CreateUser(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no responder found")
}

func TestJenkinsClient_CreateUser_WrongStatusCode(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Fcredentials%2Fstore%2Fsystem%2Fdomain%2F_%2FcreateCredentials/"+
			"credentials/store/system/domain/_/createCredentials",
		httpmock.NewStringResponder(http.StatusNotFound, str))
	platformService.On("GetSecretData", namespace, name).
		Return(secretData, nil)

	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}

	err := jc.CreateUser(instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user in Jenkins: response code:")
}

func TestJenkinsClient_CreateUser(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &jenkinsApi.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Fcredentials%2Fstore%2Fsystem%2Fdomain%2F_%2FcreateCredentials/"+
			"credentials/store/system/domain/_/createCredentials",
		httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).
		Return(secretData, nil)

	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}

	assert.NoError(t, jc.CreateUser(instance))
}

func TestJenkinsClient_GetAdminToken_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
}

func TestJenkinsClient_GetAdminToken_PostErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform POST request")
}

func TestJenkinsClient_GetAdminToken_WrongStatusCode(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process request")
}

func TestJenkinsClient_GetAdminToken_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := jc.GetAdminToken()
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

func TestJenkinsClient_GetAdminToken_NoData(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(
		http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusOK, str))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find token for admin")
}

func TestJenkinsClient_GetJobProvisions_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"
	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
}

func TestJenkinsClient_GetJobProvisions_PostErr(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to obtain Job Provisioners list")
}

func TestJenkinsClient_GetJobProvisions_WrongStatusCode(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tech script")
}

func TestJenkinsClient_GetJobProvisions_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetJobProvisions(jobPart)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

func TestJenkinsClient_GetJobProvisions_NotEqual(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"
	str2 := "{\"_class\": \"file\"}"

	httpmock.RegisterResponder(
		http.MethodGet,
		"//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(
		http.MethodPost,
		"//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusOK, str2))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a Jenkins folder")
}

func TestJenkinsClient_BuildJob_Err(t *testing.T) {
	params := map[string]string{}

	jenkins, err := createMockClient()
	assert.NoError(t, err)

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	_, err = jc.BuildJob("job", params)
	assert.Error(t, err)
}

func TestJenkinsClient_CreateFolder(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(
		http.MethodPost,
		"https://createItem",
		httpmock.NewStringResponder(http.StatusOK, ""))

	assert.NoError(t, jc.CreateFolder(name))
}

func TestJenkinsClient_CreateFolder_Err(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	httpmock.DeactivateAndReset()

	assert.Error(t, jc.CreateFolder(name))
}

func TestJenkinsClient_GetJobByName_Err(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	_, err = jc.GetJobByName(name)
	assert.Error(t, err)
}

func TestJenkinsClient_GetJobByName(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://job/name/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	_, err = jc.GetJobByName(name)
	assert.NoError(t, err)
}

func TestJenkinsClient_TriggerJob_Err(t *testing.T) {
	params := map[string]string{}

	jenkins, err := createMockClient()
	assert.NoError(t, err)

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}

	assert.Error(t, jc.TriggerJob(name, params))
}

func TestInitGoJenkinsClient(t *testing.T) {
	httpmock.Activate()
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://hostpath/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))

	ji := jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: jenkinsApi.JenkinsSpec{},
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}

	ps := pmock.PlatformService{}

	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).
		Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).
		Return(map[string][]byte{
			"username": []byte("tester"),
			"password": []byte("pwd"),
		}, nil)

	_, err := InitGoJenkinsClient(&ji, &ps)
	require.NoError(t, err)
}

func TestInitJenkinsClient(t *testing.T) {
	ji := jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: jenkinsApi.JenkinsSpec{},
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}

	ps := pmock.PlatformService{}
	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).
		Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).
		Return(map[string][]byte{
			"username": []byte("tester"),
			"password": []byte("pwd"),
		}, nil)

	_, err := InitJenkinsClient(&ji, &ps)
	require.NoError(t, err)
}
