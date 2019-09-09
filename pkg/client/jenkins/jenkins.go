package jenkins

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	"jenkins-operator/pkg/service/platform"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins_client")

// JenkinsClient abstraction fo Jenkins client
type JenkinsClient struct {
	instance        *v1alpha1.Jenkins
	PlatformService platform.PlatformService
	resty           resty.Client
}

// InitNewRestClient performs initialization of Jenkins connection
func InitJenkinsClient(instance *v1alpha1.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	route, scheme, err := platformService.GetRoute(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get route for %v", instance.Name)
	}
	apiUrl := fmt.Sprintf("%v://%v", scheme, route.Spec.Host)

	if instance.Status.AdminSecretName == "" {
		log.V(1).Info("Admin secret is not created yet")
		return nil, nil
	}

	adminSecret, err := platformService.GetSecretData(instance.Namespace, instance.Status.AdminSecretName)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get admin secret for %v", instance.Name)
	}
	jc := &JenkinsClient{
		instance:        instance,
		PlatformService: platformService,
		resty:           *resty.SetHostURL(apiUrl).SetBasicAuth(string(adminSecret["username"]), string(adminSecret["password"])),
	}
	return jc, nil
}

// InitNewRestClient performs initialization of Jenkins connection
func (jc JenkinsClient) GetCrumb() (string, error) {
	resp, err := jc.resty.R().Get("/crumbIssuer/api/json")
	if resp.StatusCode() == 404 {
		log.V(1).Info("Jenkins Crumb is not found")
		return "", nil
	}
	if err != nil || resp.IsError() {
		return "", errors.Wrap(err, "Getting Crumb failed")
	}

	var responseData map[string]string
	err = json.Unmarshal(resp.Body(), &responseData)
	if err != nil {
		return "", errors.Wrap(err, "Unmarshaling response output failed")
	}

	return responseData["crumb"], nil
}

// InitNewRestClient performs initialization of Jenkins connection
func (jc JenkinsClient) RunScript(context string) error {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	if crumb != "" {
		headers["Jenkins-Crumb"] = crumb
	}

	params := map[string]string{"script": context}
	resp, err := jc.resty.R().
		SetQueryParams(params).
		SetHeaders(headers).
		Post("/scriptText")
	if err != nil || resp.IsError() {
		return errors.Wrapf(err, fmt.Sprintf("Running script failed. Response - %s", resp.Status()))
	}
	return nil
}

func (jc JenkinsClient) CreateUser(v1alpha1.JenkinsServiceAccount) error{
	_ = createUserBody
	return nil
}

func createUserBody (instance v1alpha1.JenkinsServiceAccount) (string, error){
	switch instance.Annotations["auth-type"] {
	case "ssh":
		return createUserWithSshKey(), nil
	case "password":
		return createUserWithPassword(), nil
	default:
		return "", errors.New("Unknown authentication type!")
	}

}

func createUserWithSshKey () string {
	return ""
}

func createUserWithPassword() string {
	return ""
}


// InitNewRestClient performs initialization of Jenkins connection
func (jc JenkinsClient) GetAdminToken() (*string, error) {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	if crumb != "" {
		headers["Jenkins-Crumb"] = crumb
	}

	params := map[string]string{"newTokenName": "admin"}
	resp, err := jc.resty.R().
		SetQueryParams(params).
		SetHeaders(headers).
		Post("/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken")

	if err != nil {
		return nil, errors.Wrap(err, "Running POST request failed")
	}
	if resp.IsError() {
		return nil, errors.New(fmt.Sprintf("Request returns with error - %v", resp.Status()))
	}

	var parsedResponse map[string]interface{}
	err = json.Unmarshal(resp.Body(), &parsedResponse)
	parsedData, valid := parsedResponse["data"].(map[string]interface{})
	if valid {
		token := fmt.Sprintf("%v", parsedData["tokenValue"])
		return &token, nil
	}
	return nil, errors.New("No token find for admin")
}
