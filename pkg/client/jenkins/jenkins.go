package jenkins

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
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
)

var log = ctrl.Log.WithName("jenkins_client")

// JenkinsClient abstraction fo Jenkins client
type JenkinsClient struct {
	instance        *jenkinsApi.Jenkins
	PlatformService platform.PlatformService
	resty           *resty.Client
	GoJenkins       *gojenkins.Jenkins
}

// InitNewRestClient performs initialization of Jenkins connection
func InitJenkinsClient(instance *jenkinsApi.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	h, s, p, err := platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get route for %v", instance.Name)
	}
	apiUrl := fmt.Sprintf("%v://%v%v", s, h, p)
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
		resty:           resty.SetHostURL(apiUrl).SetBasicAuth(string(adminSecret["username"]), string(adminSecret["password"])),
	}
	return jc, nil
}

func InitGoJenkinsClient(instance *jenkinsApi.Jenkins, platformService platform.PlatformService) (*JenkinsClient, error) {
	h, shm, p, err := platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get route for %v", instance.Name)
	}

	s, err := platformService.GetSecretData(instance.Namespace, instance.Status.AdminSecretName)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get admin secret for %v", instance.Name)
	}
	url := fmt.Sprintf("%v://%v%v", shm, h, p)
	log.V(2).Info("initializing new Jenkins client", "url", url, "username", string(s["username"]))
	jenkins, err := gojenkins.CreateJenkins(http.DefaultClient, url, string(s["username"]), string(s["password"])).Init()
	if err != nil {
		return nil, err
	}
	log.Info("Jenkins client is initialized", "url", url)

	return &JenkinsClient{
		GoJenkins:       jenkins,
		PlatformService: platformService,
		resty:           resty.SetHostURL(url).SetBasicAuth(string(s["username"]), string(s["password"])),
	}, nil
}

// InitNewRestClient performs initialization of Jenkins connection
func (jc JenkinsClient) GetCrumb() (string, error) {
	resp, err := jc.resty.R().Get("/crumbIssuer/api/json")
	if err != nil {
		return "", errors.Wrap(err, "Failed to send request for Crumb!")
	}
	if resp.StatusCode() == http.StatusNotFound {
		log.V(1).Info("Jenkins Crumb is not found")
		return "", nil
	}
	if resp.IsError() {
		return "", errors.Errorf("Getting Crumb failed! Response code: %v, response body: %s", resp.StatusCode(), resp.Body())
	}

	var responseData map[string]string
	err = json.Unmarshal(resp.Body(), &responseData)
	if err != nil {
		return "", errors.Wrap(err, "Unmarshalling response output failed")
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

// GetSlaves returns a list of slaves configured in Jenkins kubernetes plugin
func (jc JenkinsClient) GetSlaves() ([]string, error) {
	c, err := jc.GetCrumb()
	if err != nil {
		return nil, err
	}
	h := make(map[string]string)
	if c != "" {
		h["Jenkins-Crumb"] = c
	}

	d, err := platformHelper.CreatePathToTemplateDirectory(defaultTechScriptsDirectory)
	if err != nil {
		return nil, err
	}
	p := fmt.Sprintf("%v/%v", d, defaultGetSlavesScript)
	cn, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, errors.Wrap(err, "Reading File err")
	}

	pr := map[string]string{"script": string(cn)}
	resp, err := jc.resty.R().
		SetQueryParams(pr).
		SetHeaders(h).
		Post("/scriptText")
	if err != nil {
		return nil, errors.Wrap(err, "Obtaining Jenkins slaves list failed!")
	}

	if resp.IsError() {
		return nil, errors.New(fmt.Sprintf("Tech script %v failed! Status: - %s", defaultGetSlavesScript, resp.Status()))
	}

	return helper.GetSlavesList(resp.String()), nil
}

// CreateUser creates new non-interactive user in Jenkins
func (jc JenkinsClient) CreateUser(instance jenkinsApi.JenkinsServiceAccount) error {
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
	if err != nil || secretData == nil {
		return errors.New(fmt.Sprintf("Couldn't get info from secret %v", instance.Spec.Credentials))
	}

	credentials, err := helper.NewJenkinsUser(secretData, instance.Spec.Type, instance.Spec.Credentials)
	if err != nil {
		return err
	}

	requestParams := map[string]string{}
	requestParams["json"], err = credentials.ToString()
	if err != nil {
		return err
	}

	resp, err := jc.resty.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10)).R().
		SetHeaders(headers).
		SetFormData(requestParams).
		Post("/credentials/store/system/domain/_/createCredentials")
	if err != nil {
		return errors.Wrap(err, "Failed to sent Jenkins user creation request!")
	}

	if resp.StatusCode() != http.StatusOK {
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
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal err. %v", resp.Body())
	}
	parsedData, valid := parsedResponse["data"].(map[string]interface{})
	if valid {
		token := fmt.Sprintf("%v", parsedData["tokenValue"])
		return &token, nil
	}
	return nil, errors.New("No token find for admin")
}

// GetJobProvisioners returns a list of Job provisioners configured in Jenkins
func (jc JenkinsClient) GetJobProvisions(jobPath string) ([]string, error) {
	var pl []string
	var raw map[string]interface{}
	c, err := jc.GetCrumb()

	if err != nil {
		return nil, err
	}
	h := make(map[string]string)
	if c != "" {
		h["Jenkins-Crumb"] = c
	}

	resp, err := jc.resty.R().
		SetHeaders(h).
		Post(fmt.Sprintf("%v/api/json?pretty=true", jobPath))
	if err != nil {
		return nil, errors.Wrap(err, "Obtaining Job Provisioners list failed!")
	}

	if resp.IsError() {
		return nil, errors.New(fmt.Sprintf("Tech script %v failed! Status: - %s", defaultGetSlavesScript, resp.Status()))
	}

	err = json.Unmarshal([]byte(resp.String()), &raw)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to unmarshal %v", []byte(resp.String()))
	}

	if raw["_class"].(string) != "com.cloudbees.hudson.plugins.folder.Folder" {
		return nil, errors.New(fmt.Sprintf("%v is not a Jenkins folder", defaultJobProvisionsFolder))
	}

	for _, p := range raw["jobs"].([]interface{}) {
		pl = append(pl, p.(map[string]interface{})["name"].(string))
	}

	return pl, nil
}

func (jc JenkinsClient) BuildJob(jobName string, parameters map[string]string) (*int64, error) {
	log.V(2).Info("start triggering job provision", "name", jobName, "codebase name", parameters["NAME"])
	qn, err := jc.GoJenkins.BuildJob(jobName, parameters)
	if qn != 0 || err != nil {
		log.V(2).Info("end triggering job provision", "name", jobName, "codebase name", parameters["NAME"])
		return jc.getBuildNumber(qn)
	}
	return nil, errors.Errorf("couldn't finish triggering job provision for %v codebase", parameters["NAME"])
}

func (jc JenkinsClient) getBuildNumber(queueNumber int64) (*int64, error) {
	log.V(2).Info("start getting build number", "queueNumber", queueNumber)
	for i := 0; i < 3; i++ {
		t, err := jc.GoJenkins.GetQueueItem(queueNumber)
		if err != nil {
			return nil, err
		}
		n := t.Raw.Executable.Number
		if n != 0 {
			log.Info("build number has been received", "number", n)
			return &n, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("couldn't get build number by queue number %v", queueNumber)
}

func (jc JenkinsClient) CreateFolder(name string) error {
	log.V(2).Info("start creating jenkins folder", "name", name)
	names, err := jc.GoJenkins.GetAllJobNames()
	if err != nil {
		return err
	}

	for _, n := range names {
		if n.Name == name {
			log.V(2).Info("Jenkins folder already exists", "name", name)
			return nil
		}
	}

	if _, err := jc.GoJenkins.CreateFolder(name); err != nil {
		return err
	}
	log.V(2).Info("end creating jenkins folder", "name", name)
	return nil
}

func (jc JenkinsClient) GetJobByName(jobName string) (*gojenkins.Job, error) {
	log.V(2).Info("start getting jenkins job", "jobName", jobName)
	job, err := jc.GoJenkins.GetJob(jobName)
	if err != nil {
		return nil, err
	}
	log.V(2).Info("end getting jenkins job", "jobName", jobName)
	return job, nil
}

func (jc JenkinsClient) TriggerJob(job string, parameters map[string]string) error {
	vLog := log.WithValues("name", job)
	vLog.Info("triggering jenkins job")
	_, err := jc.GoJenkins.BuildJob(job, parameters)
	if err != nil {
		return err
	}
	vLog.Info("jenkins job has been triggered")
	return nil
}

func (jc JenkinsClient) GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error) {
	return job.GetLastBuild()
}

func (jc JenkinsClient) BuildIsRunning(build *gojenkins.Build) bool {
	return build.IsRunning()
}
