package jenkins

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/dchest/uniuri"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1"
	gerritSpec "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	helperController "github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/helper"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	keycloakControllerHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
)

const (
	initContainerName               = "grant-permissions"
	defaultScriptsDirectory         = "scripts"
	defaultSlavesDirectory          = "slaves"
	defaultJobProvisionsDirectory   = "job-provisions"
	defaultCiJobProvisionsDirectory = "ci"
	defaultCdJobProvisionsDirectory = "cd"
	DefaultTemplatesDirectory       = "templates"
	SlavesTemplateName              = "jenkins-slaves"
	SharedLibrariesTemplateName     = "config-shared-libraries.tmpl"
	kubernetesPluginTemplateName    = "config-kubernetes-plugin.tmpl"
	keycloakConfigTemplateName      = "config-keycloak.tmpl"
	cbisTemplateName                = "cbis.json"
	jimTemplateName                 = "jim.json"
	defaultScriptConfigMapKey       = "context"
	sshKeyDefaultMountPath          = "/tmp/ssh"

	imgFolder = "img"
	jenIcon   = "jenkins.svg"

	configMapStringFormat = "%s-%s"
	pathStringFormat      = "%s/%s"
)

var log = ctrl.Log.WithName("jenkins_service")

// JenkinsService interface for Jenkins EDP component.
type JenkinsService interface {
	Configure(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	ExposeConfiguration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	Integration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	IsDeploymentReady(instance *jenkinsApi.Jenkins) (bool, error)
	CreateAdminPassword(instance *jenkinsApi.Jenkins) error
}

// NewJenkinsService function that returns JenkinsService implementation.
func NewJenkinsService(ps platform.PlatformService, k8sClient client.Client, scheme *runtime.Scheme) JenkinsService {
	return JenkinsServiceImpl{
		platformService: ps,
		k8sClient:       k8sClient,
		k8sScheme:       scheme,
		keycloakHelper:  keycloakControllerHelper.MakeHelper(k8sClient, scheme, ctrl.Log.WithName("jenkins_service")),
	}
}

// JenkinsServiceImpl struct fo Jenkins EDP Component.
type JenkinsServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
	k8sScheme       *runtime.Scheme
	keycloakHelper  *keycloakControllerHelper.Helper
}

func (j JenkinsServiceImpl) setAdminSecretInStatus(instance *jenkinsApi.Jenkins, value string) (*jenkinsApi.Jenkins, error) {
	instance.Status.AdminSecretName = value

	if err := j.k8sClient.Status().Update(context.TODO(), instance); err != nil {
		if err := j.k8sClient.Update(context.TODO(), instance); err != nil {
			return instance, fmt.Errorf("failed to set admin secret name in status: %w", err)
		}
	}

	return instance, nil
}

// newIntegrationKeycloakClient creates a v1.KeycloakClient to be used in Integration.
func (j JenkinsServiceImpl) newIntegrationKeycloakClient(instance *jenkinsApi.Jenkins) (*keycloakApi.KeycloakClient, error) {
	keycloakClient := keycloakApi.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			ClientId: instance.Name,
			Public:   !instance.Spec.KeycloakSpec.IsPrivate,
			Secret:   instance.Spec.KeycloakSpec.SecretName,
			WebUrl:   instance.Spec.ExternalURL,
			RealmRoles: &[]keycloakApi.RealmRole{
				{
					Name:      "jenkins-administrators",
					Composite: "administrator",
				},
				{
					Name:      "jenkins-users",
					Composite: "developer",
				},
			},
		},
	}

	if keycloakClient.Spec.WebUrl == "" {
		externalURL, err := j.getExternalUrl(instance)
		if err != nil {
			return nil, fmt.Errorf("failed to get route from cluster: %w", err)
		}

		keycloakClient.Spec.WebUrl = externalURL
	}

	if instance.Spec.KeycloakSpec.Realm != "" {
		keycloakClient.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
	}

	if err := j.platformService.CreateKeycloakClient(&keycloakClient); err != nil {
		return nil, fmt.Errorf("failed to create Keycloak Client data: %w", err)
	}

	keycloakClient, err := j.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak Client CR: %w", err)
	}

	return &keycloakClient, nil
}

// newIntegrationKeycloakRealm creates a v1.KeycloakRealm to be used in Integration.
func (j JenkinsServiceImpl) newIntegrationKeycloakRealm(keycloakClient *keycloakApi.KeycloakClient,
) (*keycloakApi.KeycloakRealm, error) {
	keycloakRealm, err := j.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak Realm for %s client: %w", keycloakClient.Name, err)
	}

	if keycloakRealm == nil {
		return nil, errors.New("keycloak Realm CR in not created yet")
	}

	return keycloakRealm, nil
}

// newIntegrationKeycloak creates a v1.Keycloak to be used in Integration.
func (j JenkinsServiceImpl) newIntegrationKeycloak(
	keycloakClient *keycloakApi.KeycloakClient,
	realm *keycloakApi.KeycloakRealm,
) (*keycloakApi.Keycloak, error) {
	keycloak, err := j.keycloakHelper.GetOwnerKeycloak(realm.ObjectMeta)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)

		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	if keycloak == nil {
		return nil, errors.New("keycloak CR is not created yet")
	}

	return keycloak, nil
}

// newIntegrationJenkinsScriptData creates helper.JenkinsScriptData to be used in Integration.
func (j JenkinsServiceImpl) newIntegrationJenkinsScriptData(
	instance *jenkinsApi.Jenkins,
	keycloak *keycloakApi.Keycloak,
	keycloakRealm *keycloakApi.KeycloakRealm,
	keycloakClient *keycloakApi.KeycloakClient,
) (*platformHelper.JenkinsScriptData, error) {
	jenkinsScriptData := platformHelper.JenkinsScriptData{}

	jenkinsScriptData.RealmName = keycloakRealm.Spec.RealmName
	jenkinsScriptData.KeycloakClientName = keycloakClient.Spec.ClientId
	jenkinsScriptData.KeycloakUrl = keycloak.Spec.Url
	jenkinsScriptData.KeycloakIsPrivate = instance.Spec.KeycloakSpec.IsPrivate

	if instance.Spec.KeycloakSpec.IsPrivate {
		dt, getSecretDataErr := j.platformService.GetSecretData(instance.Namespace, keycloakClient.Spec.Secret)
		if getSecretDataErr != nil {
			return nil, fmt.Errorf("failed to get keycloak client secret data: %w", getSecretDataErr)
		}

		jenkinsScriptData.KeycloakClientSecret = string(dt["clientSecret"])

		return &jenkinsScriptData, nil
	}

	return &jenkinsScriptData, nil
}

func (j JenkinsServiceImpl) mountGerritCredentials(instance *jenkinsApi.Jenkins) error {
	options := client.ListOptions{Namespace: instance.Namespace}
	list := &gerritApi.GerritList{}

	if err := j.k8sClient.List(context.TODO(), list, &options); err != nil {
		str := err.Error()

		log.Info(str)
		log.V(1).Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))

		return nil
	}

	if len(list.Items) == 0 {
		log.V(1).Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))

		return nil
	}

	gerritCrObject := &list.Items[0]
	gerritSpecName := fmt.Sprintf(pathStringFormat, gerritSpec.EdpAnnotationsPrefix, gerritSpec.EdpCiUSerSshKeySuffix)

	if val, ok := gerritCrObject.ObjectMeta.Annotations[gerritSpecName]; ok {
		volMount := []coreV1Api.VolumeMount{
			{
				Name:      val,
				MountPath: sshKeyDefaultMountPath,
				ReadOnly:  true,
			},
		}

		var mode int32 = 400

		vol := []coreV1Api.Volume{
			{
				Name: val,
				VolumeSource: coreV1Api.VolumeSource{
					Secret: &coreV1Api.SecretVolumeSource{
						SecretName:  val,
						DefaultMode: &mode,
						Items: []coreV1Api.KeyToPath{
							{
								Key:  "id_rsa",
								Path: "id_rsa",
								Mode: &mode,
							},
						},
					},
				},
			},
		}

		if err := j.platformService.AddVolumeToInitContainer(instance, initContainerName, vol, volMount); err != nil {
			return fmt.Errorf("failed to patch Jenkins DC in namespace %v: %w", instance.Namespace, err)
		}
	}

	return nil
}

func (j JenkinsServiceImpl) createSecret(
	instance *jenkinsApi.Jenkins,
	secretName, username string,
	password *string,
) error {
	var secretPassword string

	if password == nil {
		secretPassword = uniuri.New()
	} else {
		secretPassword = *password
	}

	secretData := map[string][]byte{
		"username": []byte(username),
		"password": []byte(secretPassword),
	}

	if err := j.platformService.CreateSecret(instance, secretName, secretData); err != nil {
		return fmt.Errorf("failed to create Secret %v: %w", secretName, err)
	}

	return nil
}

func setAnnotation(instance *jenkinsApi.Jenkins, key, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}

		return
	}

	instance.ObjectMeta.Annotations[key] = value
}

// Integration performs Jenkins integration with other EDP components.
func (j JenkinsServiceImpl) Integration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	if instance.Spec.KeycloakSpec.Enabled {
		enabledInstance, err := j.integrateEnabledInstance(instance)
		if err != nil {
			return enabledInstance, false, fmt.Errorf("failed to integrate Enabled Instance: %w", err)
		}

		if err = j.mountGerritCredentials(enabledInstance); err != nil {
			return enabledInstance, false, fmt.Errorf("failed to mount Gerrit credentials: %w", err)
		}

		return enabledInstance, true, nil
	}

	if err := j.mountGerritCredentials(instance); err != nil {
		return instance, false, fmt.Errorf("failed to mount Gerrit credentials: %w", err)
	}

	return instance, true, nil
}

func (j JenkinsServiceImpl) integrateEnabledInstance(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, error) {
	keycloakClient, err := j.newIntegrationKeycloakClient(instance)
	if err != nil {
		return instance, fmt.Errorf("failed to create KeycloakClient: %w", err)
	}

	keycloakRealm, err := j.newIntegrationKeycloakRealm(keycloakClient)
	if err != nil {
		return instance, fmt.Errorf("failed to create KeycloakRealm: %w", err)
	}

	keycloak, err := j.newIntegrationKeycloak(keycloakClient, keycloakRealm)
	if err != nil {
		return instance, fmt.Errorf("failed to create Keycloak: %w", err)
	}

	directoryPath, err := platformHelper.CreatePathToTemplateDirectory(DefaultTemplatesDirectory)
	if err != nil {
		return instance, fmt.Errorf("failed to create path to template directory: %w", err)
	}

	keycloakCfgFilePath := fmt.Sprintf("%s/%s", directoryPath, keycloakConfigTemplateName)

	jenkinsScriptData, err := j.newIntegrationJenkinsScriptData(instance, keycloak, keycloakRealm, keycloakClient)
	if err != nil {
		return instance, fmt.Errorf("failed to create JenkinsScriptData: %w", err)
	}

	scriptContext, err := platformHelper.ParseTemplate(jenkinsScriptData, keycloakCfgFilePath, keycloakConfigTemplateName)
	if err != nil {
		return instance, fmt.Errorf("failed to parse template: %w", err)
	}

	configKeycloakName := fmt.Sprintf(configMapStringFormat, instance.Name, "config-keycloak")
	configMapData := map[string]string{defaultScriptConfigMapKey: scriptContext.String()}

	if _, err = j.platformService.CreateConfigMap(instance, configKeycloakName, configMapData); err != nil {
		return instance, fmt.Errorf("failed to create config map: %w", err)
	}

	if _, err = j.platformService.CreateJenkinsScript(instance.Namespace, configKeycloakName, false); err != nil {
		return instance, fmt.Errorf("failed to create jenkins script: %w", err)
	}

	return instance, nil
}

// ExposeConfiguration performs exposing Jenkins configuration for other EDP components.
func (j JenkinsServiceImpl) ExposeConfiguration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	upd := false

	jc, err := jenkinsClient.InitJenkinsClient(instance, j.platformService)
	if err != nil {
		return instance, upd, fmt.Errorf("failed to init Jenkins REST client: %w", err)
	}

	if jc == nil {
		return instance, upd, errors.New("jenkins returns nil client")
	}

	sl, err := jc.GetSlaves()
	if err != nil {
		return instance, upd, fmt.Errorf("failed to get Jenkins slave list: %w", err)
	}

	ss := newSlaveArray(sl)

	if !reflect.DeepEqual(instance.Status.Slaves, ss) {
		instance.Status.Slaves = ss
		upd = true
	}

	scopes := []string{defaultCiJobProvisionsDirectory, defaultCdJobProvisionsDirectory}

	var ps []jenkinsApi.JobProvision

	for _, scope := range scopes {
		pr, getJobProvisionsErr := jc.GetJobProvisions(fmt.Sprintf("/job/%v/job/%v", defaultJobProvisionsDirectory, scope))
		if getJobProvisionsErr != nil {
			return instance, upd, fmt.Errorf("failed to get Jenkins Job provisions list for scope %v: %w", scope, getJobProvisionsErr)
		}

		for _, p := range pr {
			ps = append(ps, jenkinsApi.JobProvision{Name: p, Scope: scope})
		}
	}

	if !reflect.DeepEqual(instance.Status.JobProvisions, ps) {
		instance.Status.JobProvisions = ps
		upd = true
	}

	if err := j.createEDPComponent(instance); err != nil {
		return instance, upd, err
	}

	return instance, upd, nil
}

func newSlaveArray(slaveNames []string) []jenkinsApi.Slave {
	slaves := make([]jenkinsApi.Slave, 0, len(slaveNames))

	for _, name := range slaveNames {
		slaves = append(slaves, jenkinsApi.Slave{Name: name})
	}

	return slaves
}

func (j JenkinsServiceImpl) createEDPComponent(jen *jenkinsApi.Jenkins) error {
	url, err := j.getExternalUrl(jen)
	if err != nil {
		return err
	}

	icon, err := j.getIcon()
	if err != nil {
		return err
	}

	if err := j.platformService.CreateEDPComponentIfNotExist(jen, url, *icon); err != nil {
		return fmt.Errorf("failed to check or create EDP component: %w", err)
	}

	return nil
}

func (j JenkinsServiceImpl) getExternalUrl(jen *jenkinsApi.Jenkins) (string, error) {
	if jen.Spec.ExternalURL != "" {
		return jen.Spec.ExternalURL, nil
	}

	h, s, p, err := j.platformService.GetExternalEndpoint(jen.Namespace, jen.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get external endpoint: %w", err)
	}

	return fmt.Sprintf("%v://%v%v", s, h, p), nil
}

func (JenkinsServiceImpl) getIcon() (*string, error) {
	p, err := platformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to create path to template dir: %w", err)
	}

	filePath := fmt.Sprintf(pathStringFormat, p, jenIcon)

	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to open icon file: %w", err)
	}

	reader := bufio.NewReader(f)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read icon file: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(content)

	return &encoded, nil
}

func (j JenkinsServiceImpl) newJenkinsClient(instance *jenkinsApi.Jenkins) (*jenkinsClient.JenkinsClient, error) {
	jc, err := jenkinsClient.InitJenkinsClient(instance, j.platformService)
	if err != nil {
		return nil, fmt.Errorf("failed to init Jenkins REST client: %w", err)
	}

	if jc == nil {
		return nil, errors.New("jenkins returns nil client")
	}

	return jc, nil
}

func (j JenkinsServiceImpl) handleEmptyAdminTokenSecret(instance *jenkinsApi.Jenkins, adminTokenSecretName string,
) (*jenkinsApi.Jenkins, error) {
	jc, err := j.newJenkinsClient(instance)
	if err != nil {
		return instance, fmt.Errorf("failed to create new JenkinsClient: %w", err)
	}

	token, getAdminTokenErr := jc.GetAdminToken()
	if getAdminTokenErr != nil {
		return instance, fmt.Errorf("failed to get token from admin user: %w", getAdminTokenErr)
	}

	if err = j.createSecret(instance, adminTokenSecretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, token); err != nil {
		return instance, fmt.Errorf("failed to create secret: %w", err)
	}

	adminTokenAnnotationKey := helper.GenerateAnnotationKey(jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
	setAnnotation(instance, adminTokenAnnotationKey, adminTokenSecretName)

	if err = j.k8sClient.Update(context.TODO(), instance); err != nil {
		return instance, fmt.Errorf("failed to update jenkins instance: %w", err)
	}

	updatedInstance, setSecretErr := j.setAdminSecretInStatus(instance, adminTokenSecretName)
	if setSecretErr != nil {
		return instance, fmt.Errorf("failed to set AdminSecret in Status: %w", setSecretErr)
	}

	return updatedInstance, nil
}

func (j JenkinsServiceImpl) createScriptsFromDefaultDir(instance *jenkinsApi.Jenkins) error {
	scriptsDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(defaultScriptsDirectory)
	if err != nil {
		return fmt.Errorf("failed to create path to template dir: %w", err)
	}

	directory, err := os.ReadDir(scriptsDirectoryPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %v: %w", scriptsDirectoryPath, err)
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf(configMapStringFormat, instance.Name, file.Name())
		configMapKey := consts.JenkinsDefaultScriptConfigMapKey
		path := filepath.FromSlash(fmt.Sprintf(pathStringFormat, scriptsDirectoryPath, file.Name()))

		if err = j.createScript(instance, configMapName, configMapKey, path); err != nil {
			return fmt.Errorf("failed to create script: %w", err)
		}
	}

	return nil
}

func (j JenkinsServiceImpl) createJobProvisionsFromDefaultDir(instance *jenkinsApi.Jenkins) error {
	scopes := []string{defaultCiJobProvisionsDirectory, defaultCdJobProvisionsDirectory}

	for _, scope := range scopes {
		jobPath := fmt.Sprintf(pathStringFormat, defaultJobProvisionsDirectory, scope)

		if err := j.createJobProvisions(jobPath, instance); err != nil {
			return fmt.Errorf("failed to create JobProvisions: %w", err)
		}
	}

	return nil
}

func (j JenkinsServiceImpl) createSlavesFromDefaultDir(instance *jenkinsApi.Jenkins) error {
	slavesDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(defaultSlavesDirectory)
	if err != nil {
		return fmt.Errorf("failed to create path to template directory: %w", err)
	}

	if _, err = os.ReadDir(slavesDirectoryPath); err != nil {
		return fmt.Errorf("failed to read directory %v: %w", slavesDirectoryPath, err)
	}

	JenkinsSlavesConfigmapLabels := map[string]string{
		"role": "jenkins-slave",
	}

	if err = j.platformService.CreateConfigMapFromFileOrDir(instance, SlavesTemplateName, nil,
		slavesDirectoryPath, instance, JenkinsSlavesConfigmapLabels,
	); err != nil {
		return fmt.Errorf("failed to create config-map %v in namespace %v: %w",
			SlavesTemplateName, instance.Namespace, err)
	}

	return nil
}

func (j JenkinsServiceImpl) createTemplatesFromDefaultDir(instance *jenkinsApi.Jenkins) error {
	templatesDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(DefaultTemplatesDirectory)
	if err != nil {
		return fmt.Errorf("failed to create path to template dir: %w", err)
	}

	var templatesList []string

	templatesList = append(
		templatesList,
		SharedLibrariesTemplateName,
		kubernetesPluginTemplateName,
	)

	jenkinsScriptData := platformHelper.JenkinsScriptData{}
	jenkinsScriptData.JenkinsSharedLibraries = instance.Spec.SharedLibraries
	jenkinsScriptData.JenkinsUrl = fmt.Sprintf("http://%v:%v/%v", instance.Name, jenkinsDefaultSpec.JenkinsDefaultUiPort, instance.Spec.BasePath)

	for _, template := range templatesList {
		if err = createTemplateScript(
			templatesDirectoryPath, template,
			j.platformService,
			&jenkinsScriptData,
			instance,
		); err != nil {
			return nil
		}
	}

	for _, template := range []map[string]string{
		{"name": cbisTemplateName, "cmName": "cbis-template"},
		{"name": jimTemplateName, "cmName": "jim-template"},
	} {
		configMapName := template["cmName"]
		filePath := fmt.Sprintf("%s/%s", templatesDirectoryPath, template["name"])

		if err = j.platformService.CreateConfigMapFromFileOrDir(instance, configMapName, nil, filePath, instance); err != nil {
			return fmt.Errorf("failed to create config-map %v: %w", configMapName, err)
		}
	}

	return nil
}

// Configure performs self-configuration of Jenkins.
func (j JenkinsServiceImpl) Configure(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	adminTokenSecretName := fmt.Sprintf(configMapStringFormat, instance.Name, jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)

	adminTokenSecret, err := j.platformService.GetSecretData(instance.Namespace, adminTokenSecretName)
	if err != nil {
		return instance, false, fmt.Errorf("failed to get admin token secret for %v: %w", instance.Name, err)
	}

	if adminTokenSecret == nil {
		updatedInstance, handleErr := j.handleEmptyAdminTokenSecret(instance, adminTokenSecretName)
		if handleErr != nil {
			return updatedInstance, false, fmt.Errorf("failed to handle empty AdminTokenSecret: %w", handleErr)
		}

		instance = updatedInstance
	}

	if err = j.createScriptsFromDefaultDir(instance); err != nil {
		return instance, false, fmt.Errorf("failed to create Scripts from Default Dir: %w", err)
	}

	if err = j.createJobProvisionsFromDefaultDir(instance); err != nil {
		return instance, false, fmt.Errorf("failed to create JobProvisions from Default Dir: %w", err)
	}

	if err = j.createSlavesFromDefaultDir(instance); err != nil {
		return instance, false, fmt.Errorf("failed to create Slaves from Default Dir: %w", err)
	}

	if err = j.createTemplatesFromDefaultDir(instance); err != nil {
		return instance, false, fmt.Errorf("failed to create Templates from Default Dir: %w", err)
	}

	return instance, true, nil
}

func createTemplateScript(
	templatesDirectoryPath, template string,
	platformService platform.PlatformService,
	jenkinsScriptData *platformHelper.JenkinsScriptData,
	instance *jenkinsApi.Jenkins,
) error {
	templateFilePath := fmt.Sprintf(pathStringFormat, templatesDirectoryPath, template)

	ctx, err := platformHelper.ParseTemplate(jenkinsScriptData, templateFilePath, template)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", template, err)
	}

	jenkinsScriptName := strings.Split(template, ".")[0]
	configMapName := fmt.Sprintf(configMapStringFormat, instance.Name, jenkinsScriptName)

	configMapData := map[string]string{consts.JenkinsDefaultScriptConfigMapKey: ctx.String()}

	isUpdated, err := platformService.CreateConfigMapWithUpdate(instance, configMapName, configMapData)
	if err != nil {
		return fmt.Errorf("failed to create config map: %w", err)
	}

	if _, err = platformService.CreateJenkinsScript(instance.Namespace, configMapName, isUpdated); err != nil {
		return fmt.Errorf("failed to create jenkins script: %w", err)
	}

	return nil
}

func (j JenkinsServiceImpl) createJobProvisions(jobPath string, instance *jenkinsApi.Jenkins) error {
	jobProvisionsDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(jobPath)
	if err != nil {
		return fmt.Errorf("failed to create path to template directory: %w", err)
	}

	configMapName := strings.ReplaceAll(fmt.Sprintf(configMapStringFormat, instance.Name, jobPath), "/", "-")
	configMapKey := consts.JenkinsDefaultScriptConfigMapKey

	env, err := helperController.GetPlatformTypeEnv()
	if err != nil {
		return fmt.Errorf("failed to get platform type env: %w", err)
	}

	path := filepath.FromSlash(fmt.Sprintf(pathStringFormat, jobProvisionsDirectoryPath, env))

	if err := j.createScript(instance, configMapName, configMapKey, path); err != nil {
		return fmt.Errorf("failed to create script: %w", err)
	}

	return nil
}

// IsDeploymentReady check if DC for Jenkins is ready.
func (j JenkinsServiceImpl) IsDeploymentReady(instance *jenkinsApi.Jenkins) (bool, error) {
	res, err := j.platformService.IsDeploymentReady(instance)
	if err != nil {
		return false, fmt.Errorf("failed to check if deployment is ready: %w", err)
	}

	return res, nil
}

func (j JenkinsServiceImpl) createScript(instance *jenkinsApi.Jenkins, configMapName, configMapKey, contextPath string) error {
	jenkinsScript, err := j.platformService.CreateJenkinsScript(instance.Namespace, configMapName, false)
	if err != nil {
		return fmt.Errorf("failed to create jecnkins script: %w", err)
	}

	if err = j.platformService.CreateConfigMapFromFileOrDir(instance, configMapName, &configMapKey, contextPath, jenkinsScript); err != nil {
		return fmt.Errorf("failed to create config-map %v in namespace %v: %w", configMapName, instance.Namespace, err)
	}

	return nil
}

func (j JenkinsServiceImpl) CreateAdminPassword(instance *jenkinsApi.Jenkins) error {
	secretName := fmt.Sprintf(configMapStringFormat, instance.Name, jenkinsDefaultSpec.JenkinsAdminPasswordSuffix)

	if err := j.createSecret(instance, secretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, nil); err != nil {
		return fmt.Errorf("failed to create admin password secret: %w", err)
	}

	if instance.Status.AdminSecretName == "" {
		_, err := j.setAdminSecretInStatus(instance, secretName)
		if err != nil {
			return fmt.Errorf("failed to set admin secret in status: %w", err)
		}
	}

	return nil
}
