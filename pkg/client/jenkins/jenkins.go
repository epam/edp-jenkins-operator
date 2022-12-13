package jenkins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bndr/gojenkins"
	"gopkg.in/resty.v1"
	ctrl "sigs.k8s.io/controller-runtime"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
)

const (
	defaultTechScriptsDirectory = "tech-scripts"
	defaultGetSlavesScript      = "get-slaves"
	defaultJobProvisionsFolder  = "job-provisions"

	jenkinsCrumbKey = "Jenkins-Crumb"
	usernameKey     = "username"
	logNameKey      = "name"
	numOfAttempts   = 3
	numOfRedirects  = 10
	sleepTime       = 5 * time.Second
)

var log = ctrl.Log.WithName("jenkins_client")

// JenkinsClient abstraction fo Jenkins client.
type JenkinsClient struct {
	instance        *jenkinsApi.Jenkins
	PlatformService platform.PlatformService
	resty           *resty.Client
	GoJenkins       *gojenkins.Jenkins
}

// InitJenkinsClient performs initialization of Jenkins connection.
func InitJenkinsClient(instance *jenkinsApi.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	h, s, p, err := platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get route for %v: %w", instance.Name, err)
	}

	apiUrl := fmt.Sprintf("%v://%v%v", s, h, p)

	if instance.Status.AdminSecretName == "" {
		log.V(1).Info("Admin secret is not created yet")

		return nil, nil
	}

	adminSecret, err := platformService.GetSecretData(instance.Namespace, instance.Status.AdminSecretName)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin secret for %v: %w", instance.Name, err)
	}

	jc := &JenkinsClient{
		instance:        instance,
		PlatformService: platformService,
		resty:           resty.SetHostURL(apiUrl).SetBasicAuth(string(adminSecret[usernameKey]), string(adminSecret["password"])),
	}

	return jc, nil
}

func InitGoJenkinsClient(instance *jenkinsApi.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	h, shm, p, err := platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get route for %v: %w", instance.Name, err)
	}

	s, err := platformService.GetSecretData(instance.Namespace, instance.Status.AdminSecretName)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin secret for %v: %w", instance.Name, err)
	}

	url := fmt.Sprintf("%v://%v%v", shm, h, p)

	log.V(2).Info("initializing new Jenkins client", "url", url, usernameKey, string(s[usernameKey]))

	jenkins, err := gojenkins.CreateJenkins(http.DefaultClient, url, string(s[usernameKey]), string(s["password"])).Init()
	if err != nil {
		return nil, fmt.Errorf("failed to create jenkins: %w", err)
	}

	log.Info("Jenkins client is initialized", "url", url)

	return &JenkinsClient{
		GoJenkins:       jenkins,
		PlatformService: platformService,
		resty:           resty.SetHostURL(url).SetBasicAuth(string(s[usernameKey]), string(s["password"])),
	}, nil
}

func (jc JenkinsClient) GetCrumb() (string, error) {
	resp, err := jc.resty.R().Get("/crumbIssuer/api/json")
	if err != nil {
		return "", fmt.Errorf("failed to send request for Crumb: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.V(1).Info("Jenkins Crumb is not found")

		return "", nil
	}

	if resp.IsError() {
		return "", fmt.Errorf("failed to get crumb: response code: %v, response body: %s", resp.StatusCode(), resp.Body())
	}

	var responseData map[string]string

	if err = json.Unmarshal(resp.Body(), &responseData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response output: %w", err)
	}

	return responseData["crumb"], nil
}

// RunScript performs initialization of Jenkins connection.
func (jc JenkinsClient) RunScript(context string) error {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return err
	}

	headers := make(map[string]string)

	if crumb != "" {
		headers[jenkinsCrumbKey] = crumb
	}

	params := map[string]string{"script": context}

	resp, err := jc.resty.R().
		SetQueryParams(params).
		SetHeaders(headers).
		Post("/scriptText")
	if err != nil {
		return fmt.Errorf("failed to perform request to Jenkins script API: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to run script in Jenkins: status: %s", resp.Status())
	}

	return nil
}

// GetSlaves returns a list of slaves configured in Jenkins kubernetes plugin.
func (jc JenkinsClient) GetSlaves() ([]string, error) {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return nil, fmt.Errorf("failed to get crumb: %w", err)
	}

	headers := make(map[string]string)

	if crumb != "" {
		headers[jenkinsCrumbKey] = crumb
	}

	directory, err := platformHelper.CreatePathToTemplateDirectory(defaultTechScriptsDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create path to template dir: %w", err)
	}

	path := fmt.Sprintf("%v/%v", directory, defaultGetSlavesScript)

	cn, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read File: %w", err)
	}

	pr := map[string]string{"script": string(cn)}

	resp, err := jc.resty.R().
		SetQueryParams(pr).
		SetHeaders(headers).
		Post("/scriptText")
	if err != nil {
		return nil, fmt.Errorf("failed to obtain Jenkins slaves list: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to run tech script %v: status: %s", defaultGetSlavesScript, resp.Status())
	}

	return helper.GetSlavesList(resp.String()), nil
}

// CreateUser creates new non-interactive user in Jenkins.
func (jc JenkinsClient) CreateUser(instance *jenkinsApi.JenkinsServiceAccount) error {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	headers := make(map[string]string)

	if crumb != "" {
		headers[jenkinsCrumbKey] = crumb
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	secretData, err := jc.PlatformService.GetSecretData(instance.Namespace, instance.Spec.Credentials)
	if err != nil || secretData == nil {
		return fmt.Errorf("failed to get info from secret %v", instance.Spec.Credentials)
	}

	credentials, err := helper.NewJenkinsUser(secretData, instance.Spec.Type, instance.Spec.Credentials)
	if err != nil {
		return fmt.Errorf("failed to create new jenkins user: %w", err)
	}

	requestParams := map[string]string{}

	requestParams["json"], err = credentials.ToString()
	if err != nil {
		return fmt.Errorf("failed to parse credentials to string: %w", err)
	}

	resp, err := jc.resty.
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(numOfRedirects)).
		R().
		SetHeaders(headers).
		SetFormData(requestParams).
		Post("/credentials/store/system/domain/_/createCredentials")
	if err != nil {
		return fmt.Errorf("failed to sent Jenkins user creation request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to create user in Jenkins: response code: %v, response body: %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

func (jc JenkinsClient) GetAdminToken() (*string, error) {
	crumb, err := jc.GetCrumb()
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	if crumb != "" {
		headers[jenkinsCrumbKey] = crumb
	}

	params := map[string]string{"newTokenName": "admin"}

	resp, err := jc.resty.R().
		SetQueryParams(params).
		SetHeaders(headers).
		Post("/me/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken")
	if err != nil {
		return nil, fmt.Errorf("failed to perform POST request: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to process request: returns error: %v", resp.Status())
	}

	var parsedResponse map[string]interface{}

	if err = json.Unmarshal(resp.Body(), &parsedResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Jenkins response %v: %w", resp.Body(), err)
	}

	parsedData, valid := parsedResponse["data"].(map[string]interface{})
	if valid {
		token := fmt.Sprintf("%v", parsedData["tokenValue"])

		return &token, nil
	}

	return nil, errors.New("failed to find token for admin")
}

// GetJobProvisions returns a list of Job provisions configured in Jenkins.
func (jc JenkinsClient) GetJobProvisions(jobPath string) ([]string, error) {
	var provisionNames []string

	raw, err := jc.obtainRawJobProvisions(jobPath)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain raw JobProvisioners data: %w", err)
	}

	classValue, ok := raw["_class"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to access \"_class\" field")
	}

	if classValue != "com.cloudbees.hudson.plugins.folder.Folder" {
		return nil, fmt.Errorf("failed to get job provisions: %v is not a Jenkins folder", defaultJobProvisionsFolder)
	}

	jobValues, ok := raw["jobs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to access \"jobs\" field")
	}

	for _, jobProvision := range jobValues {
		provisionName, ok := jobProvision.(map[string]interface{})["name"].(string)
		if !ok {
			return nil, fmt.Errorf("failed to access \"name\" field of a job")
		}

		provisionNames = append(provisionNames, provisionName)
	}

	return provisionNames, nil
}

func (jc JenkinsClient) obtainRawJobProvisions(jobPath string) (map[string]interface{}, error) {
	rawJobProvisioners := make(map[string]interface{})

	crumb, err := jc.GetCrumb()
	if err != nil {
		return nil, fmt.Errorf("failed to get crumb: %w", err)
	}

	headers := make(map[string]string)
	if crumb != "" {
		headers[jenkinsCrumbKey] = crumb
	}

	resp, err := jc.resty.
		R().
		SetHeaders(headers).
		Post(fmt.Sprintf("%v/api/json?pretty=true", jobPath))
	if err != nil {
		return nil, fmt.Errorf("failed to obtain Job Provisioners list: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to run tech script %v: status: %s", defaultGetSlavesScript, resp.Status())
	}

	if err = json.Unmarshal([]byte(resp.String()), &rawJobProvisioners); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JobProvisioners %s: %w", resp.String(), err)
	}

	return rawJobProvisioners, nil
}

func (jc JenkinsClient) BuildJob(jobName string, parameters map[string]string) (*int64, error) {
	log.V(2).Info("start triggering job provision", logNameKey, jobName, "codebase name", parameters["NAME"])

	qn, err := jc.GoJenkins.BuildJob(jobName, parameters)
	if qn != 0 || err != nil {
		log.V(2).Info("end triggering job provision", logNameKey, jobName, "codebase name", parameters["NAME"])

		return jc.getBuildNumber(qn)
	}

	return nil, fmt.Errorf("failed to finish triggering job provision for %v codebase", parameters["NAME"])
}

func (jc JenkinsClient) getBuildNumber(queueNumber int64) (*int64, error) {
	log.V(2).Info("start getting build number", "queueNumber", queueNumber)

	for i := 0; i < numOfAttempts; i++ {
		t, err := jc.GoJenkins.GetQueueItem(queueNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get queue item: %w", err)
		}

		n := t.Raw.Executable.Number
		if n != 0 {
			log.Info("build number has been received", "number", n)

			return &n, nil
		}

		time.Sleep(sleepTime)
	}

	return nil, fmt.Errorf("failed to get build number by queue number %v", queueNumber)
}

func (jc JenkinsClient) CreateFolder(name string) error {
	log.V(2).Info("start creating jenkins folder", logNameKey, name)

	names, err := jc.GoJenkins.GetAllJobNames()
	if err != nil {
		return fmt.Errorf("failed to GetAllJobNames: %w", err)
	}

	for _, n := range names {
		if n.Name == name {
			log.V(2).Info("Jenkins folder already exists", logNameKey, name)

			return nil
		}
	}

	if _, err := jc.GoJenkins.CreateFolder(name); err != nil {
		return fmt.Errorf("failed to CreateFolder: %w", err)
	}

	log.V(2).Info("end creating jenkins folder", logNameKey, name)

	return nil
}

func (jc JenkinsClient) GetJobByName(jobName string) (*gojenkins.Job, error) {
	log.V(2).Info("start getting jenkins job", "jobName", jobName)

	job, err := jc.GoJenkins.GetJob(jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to GetJob: %w", err)
	}

	log.V(2).Info("end getting jenkins job", "jobName", jobName)

	return job, nil
}

func (jc JenkinsClient) TriggerJob(job string, parameters map[string]string) error {
	vLog := log.WithValues(logNameKey, job)

	vLog.Info("triggering jenkins job")

	if _, err := jc.GoJenkins.BuildJob(job, parameters); err != nil {
		return fmt.Errorf("failed to BuildJob: %w", err)
	}

	vLog.Info("jenkins job has been triggered")

	return nil
}

func (JenkinsClient) GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error) {
	build, err := job.GetLastBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to GetLastBuild form the job: %w", err)
	}

	return build, nil
}

func (JenkinsClient) BuildIsRunning(build *gojenkins.Build) bool {
	return build.IsRunning()
}
