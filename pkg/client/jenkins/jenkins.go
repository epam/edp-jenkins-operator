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

	if instance.Status.AdminSecretName == nil {
		log.V(1).Info("Admin secret is not created yet")
		return nil, nil
	}

	adminSecret, err := platformService.GetSecretData(instance.Namespace, *instance.Status.AdminSecretName)
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
	var responseData map[string]string
	err = json.Unmarshal(resp.Body(), &responseData)
	if resp.StatusCode() == 404 {
		return "", nil
	}

	if err != nil || resp.IsError() {
		return "", errors.Wrap(err, "Getting Crumb failed")
	}
	return responseData["crumb"], nil
}

// InitNewRestClient performs initialization of Jenkins connection
func (jc JenkinsClient) RunScript(context string) error {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return errors.Wrapf(err, "Failed to get crumb")
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
