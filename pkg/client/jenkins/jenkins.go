package jenkins

import (
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
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
	host, scheme, err := platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get route for %v", instance.Name)
	}
	apiUrl := fmt.Sprintf("%v://%v", scheme, host)
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
	if err != nil {
		return "", errors.Wrap(err, "Failed to send request for Crumb!")
	}
	if resp.StatusCode() == 404 {
		log.V(1).Info("Jenkins Crumb is not found")
		return "", nil
	}
	if resp.IsError() {
		return "", errors.Wrapf(err, "Getting Crumb failed! Response code: %v, response body: %s", resp.StatusCode(), resp.Body())
	}

	var responseData map[string]string
	err = json.Unmarshal(resp.Body(), &responseData)
	if err != nil {
		return "", errors.Wrap(err, "Unmarshaling response output failed")
	}

	return responseData["crumb"], nil
}

// RunScript performs initialization of Jenkins connection
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
	if err != nil {
		return errors.Wrap(err, "Request to Jenkins script API failed!")
	}

	if resp.IsError() {
		return errors.New(fmt.Sprintf("Running script in Jenkins failed! Status: - %s", resp.Status()))
	}

	return nil
}

// CreateUser creates new non-interactive user in Jenkins
func (jc JenkinsClient) CreateUser(instance v1alpha1.JenkinsServiceAccount) error {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	if crumb != "" {
		headers["Jenkins-Crumb"] = crumb
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	secretData, err := jc.PlatformService.GetSecretData(instance.Namespace, instance.Spec.Credentials)
	if err != nil {
		return err
	}

	credentials, err := helper.NewJenkinsUser(secretData, instance.Spec.Type)
	if err != nil {
		return err
	}

	requestParams := map[string]string{}
	requestParams["json"], err = credentials.ToString()
	if err != nil {
		return err
	}

	resp, err := jc.resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(10)).R().
		SetQueryParams(requestParams).
		SetHeaders(headers).
		Post("/credentials/store/system/domain/_/createCredentials")
	if err != nil {
		return errors.Wrap(err, "Failed to sent Jenkins user creation request!")
	}

	if resp.StatusCode() != 200 {
		return errors.New(fmt.Sprintf("Failed to create user in Jenkins! Response code: %v, response body: %s", resp.StatusCode(), resp.Body()))
	}

	return nil
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
