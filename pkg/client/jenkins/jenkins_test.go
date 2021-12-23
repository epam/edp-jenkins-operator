package jenkins

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	name      = "name"
	namespace = "namespace"
	str       = "{\"crumb\": \"file\"}"
)

func createMockClient() (*gojenkins.Jenkins, error) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
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
	instance := &v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	errTest := errors.New("test")
	platformService.On("GetExternalEndpoint", namespace, name).Return("", "", "", errTest)
	_, err := InitJenkinsClient(instance, &platformService)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unable to get route for"))
	platformService.AssertExpectations(t)
}

func TestInitJenkinsClient_EmptySecretName(t *testing.T) {
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	instance.Status.AdminSecretName = ""

	platformService.On("GetExternalEndpoint", namespace, name).Return("", "", "", nil)
	_, err := InitJenkinsClient(instance, &platformService)
	assert.NoError(t, err)
	platformService.AssertExpectations(t)
}

func TestInitJenkinsClient_GetSecretDataErr(t *testing.T) {
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	instance.Status.AdminSecretName = name
	errTest := errors.New("test")
	platformService.On("GetExternalEndpoint", namespace, name).Return("", "", "", nil)
	platformService.On("GetSecretData", namespace, name).Return(nil, errTest)
	_, err := InitJenkinsClient(instance, &platformService)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unable to get admin secret for"))
	platformService.AssertExpectations(t)
}

func TestJenkinsClient_GetCrumb_GetErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}
	crumb, err := jc.GetCrumb()
	assert.Error(t, err)
	assert.Equal(t, "", crumb)
}

func TestJenkinsClient_GetCrumb_NoFoundCode(t *testing.T) {
	restyClient := CreateMockResty()
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusNotFound, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}
	crumb, err := jc.GetCrumb()
	assert.NoError(t, err)
	assert.Equal(t, "", crumb)
}

func TestJenkinsClient_GetCrumb_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}
	crumb, err := jc.GetCrumb()
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
	assert.Equal(t, "", crumb)
}

func TestJenkinsClient_GetCrumb(t *testing.T) {
	restyClient := CreateMockResty()
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))

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
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusBadGateway, ""))
	jc := JenkinsClient{
		resty: restyClient,
	}
	err := jc.RunScript(script)
	assert.Error(t, err)
}

func TestJenkinsClient_RunScript_PostErr(t *testing.T) {
	restyClient := CreateMockResty()
	script := "test"
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	jc := JenkinsClient{
		resty: restyClient,
	}
	err := jc.RunScript(script)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Request to Jenkins script API failed!"))
}

func TestJenkinsClient_RunScript_NotFoundCode(t *testing.T) {
	restyClient := CreateMockResty()
	script := "test"
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "//%2FscriptText%3Fscript="+script+"/scriptText?script="+script, httpmock.NewStringResponder(http.StatusNotFound, ""))
	jc := JenkinsClient{
		resty: restyClient,
	}
	err := jc.RunScript(script)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Running script in Jenkins failed!"))
}

func TestJenkinsClient_RunScript(t *testing.T) {
	restyClient := CreateMockResty()
	script := "test"
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "//%2FscriptText%3Fscript="+script+"/scriptText?script="+script, httpmock.NewStringResponder(http.StatusOK, ""))
	jc := JenkinsClient{
		resty: restyClient,
	}
	err := jc.RunScript(script)
	assert.NoError(t, err)
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
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	jc := JenkinsClient{
		resty: restyClient,
	}
	_, err := jc.GetSlaves()
	assert.Error(t, err)
}

func TestJenkinsClient_CreateUser_GetCrumbErr(t *testing.T) {
	restyClient := CreateMockResty()
	instance := &v1alpha1.JenkinsServiceAccount{}
	jc := JenkinsClient{
		resty: restyClient,
	}
	err := jc.CreateUser(*instance)
	assert.Error(t, err)
}

func TestJenkinsClient_CreateUser_GetSecretDataErr(t *testing.T) {
	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	errTest := errors.New("test")
	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).Return(nil, errTest)
	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}
	err := jc.CreateUser(*instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Couldn't get info from secret"))
}

func TestJenkinsClient_CreateUser_NewJenkinsUserErr(t *testing.T) {
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).Return(secretData, nil)
	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}
	err := jc.CreateUser(*instance)
	assert.Error(t, err)
}

func TestJenkinsClient_CreateUser_PostErr(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).Return(secretData, nil)
	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}
	err := jc.CreateUser(*instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no responder found"))
}

func TestJenkinsClient_CreateUser_WrongStatusCode(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost,
		"//%2Fcredentials%2Fstore%2Fsystem%2Fdomain%2F_%2FcreateCredentials/credentials/store/system/domain/_/createCredentials",
		httpmock.NewStringResponder(http.StatusNotFound, str))
	platformService.On("GetSecretData", namespace, name).Return(secretData, nil)
	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}
	err := jc.CreateUser(*instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to create user in Jenkins! Response code:"))
}

func TestJenkinsClient_CreateUser(t *testing.T) {
	secretData := map[string][]byte{}

	restyClient := CreateMockResty()
	platformService := pmock.PlatformService{}
	instance := &v1alpha1.JenkinsServiceAccount{}
	instance.Namespace = namespace
	instance.Spec.Credentials = name
	instance.Spec.Type = "ssh"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost,
		"//%2Fcredentials%2Fstore%2Fsystem%2Fdomain%2F_%2FcreateCredentials/credentials/store/system/domain/_/createCredentials",
		httpmock.NewStringResponder(http.StatusOK, str))
	platformService.On("GetSecretData", namespace, name).Return(secretData, nil)
	jc := JenkinsClient{
		resty:           restyClient,
		PlatformService: &platformService,
	}
	err := jc.CreateUser(*instance)
	assert.NoError(t, err)
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

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Running POST request failed"))
}

func TestJenkinsClient_GetAdminToken_WrongStatusCode(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Request returns with error"))
}

func TestJenkinsClient_GetAdminToken_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := jc.GetAdminToken()
	errJson := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJson)
}

func TestJenkinsClient_GetAdminToken_NoData(t *testing.T) {
	restyClient := CreateMockResty()
	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost,
		"//%2Fme%2FdescriptorByName%2Fjenkins.security.ApiTokenProperty%2FgenerateNewToken%3FnewTokenName"+
			"=admin/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken?newTokenName=admin",
		httpmock.NewStringResponder(http.StatusOK, str))

	_, err := jc.GetAdminToken()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "No token find for admin"))
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

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))

	jc := JenkinsClient{
		resty: restyClient,
	}
	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Obtaining Job Provisioners list failed!"))
}

func TestJenkinsClient_GetJobProvisions_WrongStatusCode(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}
	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Tech script"))
}

func TestJenkinsClient_GetJobProvisions_UnmarshalErr(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}
	_, err := jc.GetJobProvisions(jobPart)
	errJSON := &json.SyntaxError{}
	assert.ErrorAs(t, err, &errJSON)
}

func TestJenkinsClient_GetJobProvisions_NotEqual(t *testing.T) {
	restyClient := CreateMockResty()
	jobPart := "test"
	str2 := "{\"_class\": \"file\"}"

	httpmock.RegisterResponder(http.MethodGet, "//%2FcrumbIssuer%2Fapi%2Fjson/crumbIssuer/api/json",
		httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "//%2Ftest%2Fapi%2Fjson%3Fpretty=true/test/api/json?pretty=true",
		httpmock.NewStringResponder(http.StatusOK, str2))

	jc := JenkinsClient{
		resty: restyClient,
	}
	_, err := jc.GetJobProvisions(jobPart)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "is not a Jenkins folder"))
}

func TestJenkinsClient_IsBuildSuccessful_GetJobErr(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))

	jc := JenkinsClient{
		GoJenkins: jenkins,
	}
	_, err = jc.IsBuildSuccessful("", int64(1))
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "could't get job "))
}

func TestJenkinsClient_IsBuildSuccessful_getBuildErr(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	jc := JenkinsClient{
		GoJenkins: jenkins,
	}
	_, err = jc.IsBuildSuccessful("", int64(1))
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "could't get build"))
}

func TestJenkinsClient_IsBuildSuccessful_getBuildNotFound(t *testing.T) {
	jenkins, err := createMockClient()

	assert.NoError(t, err)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job/api/json", httpmock.NewStringResponder(http.StatusNotFound, ""))
	jc := JenkinsClient{
		GoJenkins: jenkins,
	}
	successful, err := jc.IsBuildSuccessful("", int64(1))
	if err != nil {
		return
	}
	assert.NoError(t, err)
	assert.False(t, successful)
}

func TestJenkinsClient_IsBuildSuccessful(t *testing.T) {
	jenkins, err := createMockClient()
	httpmock.RegisterResponder(http.MethodGet, "https://job/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://job//1/api/json?depth=1", httpmock.NewStringResponder(http.StatusOK, ""))

	assert.NoError(t, err)
	jc := JenkinsClient{
		GoJenkins: jenkins,
	}
	_, err = jc.IsBuildSuccessful("", int64(1))
	assert.NoError(t, err)
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

func TestJenkinsClient_CreateFolder_Err(t *testing.T) {
	jenkins, err := createMockClient()
	assert.NoError(t, err)
	jc := JenkinsClient{
		GoJenkins: jenkins,
	}
	httpmock.DeactivateAndReset()
	err = jc.CreateFolder(name)
	assert.Error(t, err)
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

	httpmock.RegisterResponder(http.MethodGet, "https://job/name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))

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
	err = jc.TriggerJob(name, params)
	assert.Error(t, err)
}

func TestInitGoJenkinsClient(t *testing.T) {
	httpmock.Activate()
	httpmock.RegisterResponder(http.MethodGet, "http://hostpath/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""))

	ji := v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: v1alpha1.JenkinsSpec{},
		Status: v1alpha1.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}
	ps := platform.Mock{}

	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).Return(map[string][]byte{
		"username": []byte("tester"),
		"password": []byte("pwd"),
	}, nil)
	if _, err := InitGoJenkinsClient(&ji, &ps); err != nil {
		t.Fatal(err)
	}
}

func TestInitJenkinsClient(t *testing.T) {
	ji := v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: v1alpha1.JenkinsSpec{},
		Status: v1alpha1.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}
	ps := platform.Mock{}
	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).Return(map[string][]byte{
		"username": []byte("tester"),
		"password": []byte("pwd"),
	}, nil)

	if _, err := InitJenkinsClient(&ji, &ps); err != nil {
		t.Fatal(err)
	}
}
