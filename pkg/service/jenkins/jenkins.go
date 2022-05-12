package jenkins

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/dchest/uniuri"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritSpec "github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakControllerHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	helperController "github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/helper"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
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
)

var log = ctrl.Log.WithName("jenkins_service")

// JenkinsService interface for Jenkins EDP component
type JenkinsService interface {
	Configure(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	ExposeConfiguration(instance jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	Integration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error)
	IsDeploymentReady(instance jenkinsApi.Jenkins) (bool, error)
	CreateAdminPassword(instance *jenkinsApi.Jenkins) error
}

// NewJenkinsService function that returns JenkinsService implementation
func NewJenkinsService(ps platform.PlatformService, client client.Client, scheme *runtime.Scheme) JenkinsService {
	return JenkinsServiceImpl{
		platformService: ps,
		k8sClient:       client,
		k8sScheme:       scheme,
		keycloakHelper:  keycloakControllerHelper.MakeHelper(client, scheme, ctrl.Log.WithName("jenkins_service")),
	}
}

// JenkinsServiceImpl struct fo Jenkins EDP Component
type JenkinsServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
	k8sScheme       *runtime.Scheme
	keycloakHelper  *keycloakControllerHelper.Helper
}

func (j JenkinsServiceImpl) setAdminSecretInStatus(instance *jenkinsApi.Jenkins, value string) (*jenkinsApi.Jenkins, error) {
	instance.Status.AdminSecretName = value
	err := j.k8sClient.Status().Update(context.TODO(), instance)
	if err != nil {
		err := j.k8sClient.Update(context.TODO(), instance)
		if err != nil {
			return instance, errors.Wrap(err, "Couldn't set admin secret name in status")
		}
	}
	return instance, nil
}

func (j JenkinsServiceImpl) createKeycloakClient(instance jenkinsApi.Jenkins, name string) (*keycloakApi.KeycloakClient, error) {
	keycloakClientObject := &keycloakApi.KeycloakClient{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1.edp.epam.com/v1alpha1",
			Kind:       "KeycloakClient",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			TargetRealm: instance.Spec.KeycloakSpec.Realm,
			Public:      true,
			ClientId:    instance.Name,
		},
	}

	nsn := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      name,
	}

	err := j.k8sClient.Get(context.TODO(), nsn, keycloakClientObject)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err := j.k8sClient.Create(context.TODO(), keycloakClientObject)
			if err != nil {
				return nil, errors.Wrapf(err, "Couldn't create Keycloak client object %v", name)
			}
			log.Info("Keycloak client CR created")
			return keycloakClientObject, nil
		}
		return nil, errors.Wrapf(err, "Couldn't get Keycloak client object %v", name)
	}

	return keycloakClientObject, nil
}

func (j JenkinsServiceImpl) mountGerritCredentials(instance *jenkinsApi.Jenkins) error {
	options := client.ListOptions{Namespace: instance.Namespace}
	list := &gerritApi.GerritList{}

	err := j.k8sClient.List(context.TODO(), list, &options)
	if err != nil {
		str := err.Error()
		fmt.Println(str)
		log.V(1).Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))
		return nil
	}

	if len(list.Items) == 0 {
		log.V(1).Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))
		return nil
	}

	gerritCrObject := &list.Items[0]
	gerritSpecName := fmt.Sprintf("%v/%v", gerritSpec.EdpAnnotationsPrefix, gerritSpec.EdpCiUSerSshKeySuffix)
	if val, ok := gerritCrObject.ObjectMeta.Annotations[gerritSpecName]; ok {
		volMount := []coreV1Api.VolumeMount{
			{
				Name:      val,
				MountPath: sshKeyDefaultMountPath,
				ReadOnly:  true,
			},
		}

		mode := int32(400)
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

		err = j.platformService.AddVolumeToInitContainer(instance, initContainerName, vol, volMount)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Unable to patch Jenkins DC in namespace %v", instance.Namespace))
		}
	}
	return nil
}

func (j JenkinsServiceImpl) createSecret(instance *jenkinsApi.Jenkins, secretName string, username string, password *string) error {
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

	err := j.platformService.CreateSecret(instance, secretName, secretData)
	if err != nil {
		return errors.Wrapf(err, "Failed to create Secret %v", secretName)
	}
	return nil
}

//setAnnotation add key:value to current resource annotation
func setAnnotation(instance *jenkinsApi.Jenkins, key string, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}

// Integration performs integration Jenkins with other EDP components
func (j JenkinsServiceImpl) Integration(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	if instance.Spec.KeycloakSpec.Enabled {

		h, s, p, err := j.platformService.GetExternalEndpoint(instance.Namespace, instance.Name)
		if err != nil {
			return instance, false, errors.Wrap(err, "Failed to get route from cluster!")
		}

		webUrl := fmt.Sprintf("%v://%v%v", s, h, p)
		keycloakClient := keycloakV1Api.KeycloakClient{}
		keycloakClient.Name = instance.Name
		keycloakClient.Namespace = instance.Namespace
		keycloakClient.Spec.ClientId = instance.Name
		keycloakClient.Spec.Public = !instance.Spec.KeycloakSpec.IsPrivate
		keycloakClient.Spec.Secret = instance.Spec.KeycloakSpec.SecretName
		keycloakClient.Spec.WebUrl = webUrl
		keycloakClient.Spec.RealmRoles = &[]keycloakV1Api.RealmRole{
			{
				Name:      "jenkins-administrators",
				Composite: "administrator",
			},
			{
				Name:      "jenkins-users",
				Composite: "developer",
			},
		}
		if instance.Spec.KeycloakSpec.Realm != "" {
			keycloakClient.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
		}

		err = j.platformService.CreateKeycloakClient(&keycloakClient)
		if err != nil {
			return instance, false, errors.Wrap(err, "Failed to create Keycloak Client data!")
		}

		keycloakClient, err = j.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return instance, false, errors.Wrap(err, "Failed to get Keycloak Client CR!")
		}

		keycloakRealm, err := j.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Failed to get Keycloak Realm for %s client!", keycloakClient.Name)
		}

		if keycloakRealm == nil {
			return instance, false, errors.New("Keycloak Realm CR in not created yet!")
		}

		keycloak, err := j.keycloakHelper.GetOwnerKeycloak(keycloakRealm.ObjectMeta)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)
			return instance, false, errors.Wrap(err, errMsg)
		}

		if keycloak == nil {
			return instance, false, errors.New("Keycloak CR is not created yet!")
		}

		directoryPath, err := platformHelper.CreatePathToTemplateDirectory(DefaultTemplatesDirectory)
		if err != nil {
			return instance, false, errors.Wrap(err, "unable to create path to template directory")
		}
		keycloakCfgFilePath := fmt.Sprintf("%s/%s", directoryPath, keycloakConfigTemplateName)

		jenkinsScriptData := platformHelper.JenkinsScriptData{}
		jenkinsScriptData.RealmName = keycloakRealm.Spec.RealmName
		jenkinsScriptData.KeycloakClientName = keycloakClient.Spec.ClientId
		jenkinsScriptData.KeycloakUrl = keycloak.Spec.Url
		jenkinsScriptData.KeycloakIsPrivate = instance.Spec.KeycloakSpec.IsPrivate

		if instance.Spec.KeycloakSpec.IsPrivate {
			dt, err := j.platformService.GetSecretData(instance.Namespace, keycloakClient.Spec.Secret)
			if err != nil {
				return instance, false, errors.Wrap(err, "unable to get keycloak client secret data")
			}

			jenkinsScriptData.KeycloakClientSecret = string(dt["clientSecret"])
		}

		scriptContext, err := platformHelper.ParseTemplate(jenkinsScriptData, keycloakCfgFilePath, keycloakConfigTemplateName)
		if err != nil {
			return instance, false, err
		}

		configKeycloakName := fmt.Sprintf("%v-%v", instance.Name, "config-keycloak")
		configMapData := map[string]string{defaultScriptConfigMapKey: scriptContext.String()}
		_, err = j.platformService.CreateConfigMap(instance, configKeycloakName, configMapData)
		if err != nil {
			return instance, false, err
		}

		_, err = j.platformService.CreateJenkinsScript(instance.Namespace, configKeycloakName, false)
		if err != nil {
			return instance, false, err
		}
	}

	err := j.mountGerritCredentials(instance)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Failed to mount Gerrit credentials")
	}

	return instance, true, nil
}

// ExposeConfiguration performs exposing Jenkins configuration for other EDP components
func (j JenkinsServiceImpl) ExposeConfiguration(instance jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	upd := false

	jc, err := jenkinsClient.InitJenkinsClient(&instance, j.platformService)
	if err != nil {
		return &instance, upd, errors.Wrap(err, "Failed to init Jenkins REST client")
	}
	if jc == nil {
		return &instance, upd, errors.New("Jenkins returns nil client")
	}

	sl, err := jc.GetSlaves()
	if err != nil {
		return &instance, upd, errors.Wrapf(err, "Unable to get Jenkins slaves list")
	}

	ss := []jenkinsApi.Slave{}
	for _, s := range sl {
		ss = append(ss, jenkinsApi.Slave{Name: s})
	}

	if !reflect.DeepEqual(instance.Status.Slaves, ss) {
		instance.Status.Slaves = ss
		upd = true
	}

	scopes := []string{defaultCiJobProvisionsDirectory, defaultCdJobProvisionsDirectory}
	ps := []jenkinsApi.JobProvision{}
	for _, scope := range scopes {
		pr, err := jc.GetJobProvisions(fmt.Sprintf("/job/%v/job/%v", defaultJobProvisionsDirectory, scope))
		if err != nil {
			return &instance, upd, errors.Wrapf(err, "Unable to get Jenkins Job provisions list for scope %v", scope)
		}
		for _, p := range pr {
			ps = append(ps, jenkinsApi.JobProvision{Name: p, Scope: scope})
		}
	}

	if !reflect.DeepEqual(instance.Status.JobProvisions, ps) {
		instance.Status.JobProvisions = ps
		upd = true
	}
	err = j.createEDPComponent(instance)
	return &instance, upd, err
}

func (j JenkinsServiceImpl) createEDPComponent(jen jenkinsApi.Jenkins) error {
	url, err := j.getUrl(jen)
	if err != nil {
		return err
	}
	icon, err := j.getIcon()
	if err != nil {
		return err
	}
	return j.platformService.CreateEDPComponentIfNotExist(jen, *url, *icon)
}

func (j JenkinsServiceImpl) getUrl(jen jenkinsApi.Jenkins) (*string, error) {
	h, s, p, err := j.platformService.GetExternalEndpoint(jen.Namespace, jen.Name)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%v://%v%v", s, h, p)
	return &url, nil
}

func (j JenkinsServiceImpl) getIcon() (*string, error) {
	p, err := platformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, err
	}
	fp := fmt.Sprintf("%v/%v", p, jenIcon)
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}

// Configure performs self-configuration of Jenkins
func (j JenkinsServiceImpl) Configure(instance *jenkinsApi.Jenkins) (*jenkinsApi.Jenkins, bool, error) {
	jc, err := jenkinsClient.InitJenkinsClient(instance, j.platformService)
	if err != nil {
		return instance, false, errors.Wrap(err, "Failed to init Jenkins REST client")
	}
	if jc == nil {
		return instance, false, errors.New("Jenkins returns nil client")
	}

	adminTokenSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
	adminTokenSecret, err := j.platformService.GetSecretData(instance.Namespace, adminTokenSecretName)
	if err != nil {
		return instance, false, errors.Wrapf(err, "Unable to get admin token secret for %v", instance.Name)
	}

	if adminTokenSecret == nil {
		token, err := jc.GetAdminToken()
		if err != nil {
			return instance, false, errors.Wrap(err, "Failed to get token from admin user")
		}

		err = j.createSecret(instance, adminTokenSecretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, token)
		if err != nil {
			return instance, false, err
		}

		adminTokenAnnotationKey := helper.GenerateAnnotationKey(jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
		setAnnotation(instance, adminTokenAnnotationKey, adminTokenSecretName)

		err = j.k8sClient.Update(context.TODO(), instance)
		if err != nil {
			return instance, false, err
		}

		updatedInstance, err := j.setAdminSecretInStatus(instance, adminTokenSecretName)
		if err != nil {
			return instance, false, err
		}
		instance = updatedInstance
	}

	scriptsDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(defaultScriptsDirectory)
	if err != nil {
		return instance, false, err
	}

	directory, err := ioutil.ReadDir(scriptsDirectoryPath)
	if err != nil {
		return instance, false, errors.Wrapf(err, fmt.Sprintf("Couldn't read directory %v", scriptsDirectoryPath))
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, file.Name())
		configMapKey := consts.JenkinsDefaultScriptConfigMapKey

		path := filepath.FromSlash(fmt.Sprintf("%v/%v", scriptsDirectoryPath, file.Name()))
		err = j.createScript(instance, configMapName, configMapKey, path)
		if err != nil {
			return instance, false, err
		}
	}

	scopes := []string{defaultCiJobProvisionsDirectory, defaultCdJobProvisionsDirectory}
	for _, scope := range scopes {
		err = j.createJobProvisions(fmt.Sprintf("%v/%v", defaultJobProvisionsDirectory, scope), jc, instance)
		if err != nil {
			return instance, false, err
		}
	}

	slavesDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(defaultSlavesDirectory)
	if err != nil {
		return instance, false, err
	}

	_, err = ioutil.ReadDir(slavesDirectoryPath)
	if err != nil {
		return nil, false, errors.Wrapf(err, fmt.Sprintf("Couldn't read directory %v", slavesDirectoryPath))
	}

	JenkinsSlavesConfigmapLabels := map[string]string{
		"role": "jenkins-slave",
	}

	err = j.platformService.CreateConfigMapFromFileOrDir(instance, SlavesTemplateName, nil,
		slavesDirectoryPath, instance, JenkinsSlavesConfigmapLabels)
	if err != nil {
		return nil, false, errors.Wrapf(err, "Couldn't create configs-map %v in namespace %v.",
			SlavesTemplateName, instance.Namespace)
	}

	templatesDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(DefaultTemplatesDirectory)
	if err != nil {
		return instance, false, err
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
		if err := createTemplateScript(templatesDirectoryPath, template, j.platformService, jenkinsScriptData,
			instance); err != nil {
			return instance, false, nil
		}
	}

	for _, template := range []map[string]string{
		{"name": cbisTemplateName, "cmName": "cbis-template"},
		{"name": jimTemplateName, "cmName": "jim-template"},
	} {
		configMapName := template["cmName"]
		filePath := fmt.Sprintf("%s/%s", templatesDirectoryPath, template["name"])
		err = j.platformService.CreateConfigMapFromFileOrDir(instance, configMapName, nil, filePath, instance)
		if err != nil {
			return instance, false, errors.Wrapf(err, "Couldn't create config-map %v", configMapName)
		}
	}

	return instance, true, nil
}

func createTemplateScript(templatesDirectoryPath, template string, platformService platform.PlatformService,
	jenkinsScriptData platformHelper.JenkinsScriptData, instance *jenkinsApi.Jenkins) error {
	templateFilePath := fmt.Sprintf("%v/%v", templatesDirectoryPath, template)
	ctx, err := platformHelper.ParseTemplate(jenkinsScriptData, templateFilePath, template)
	if err != nil {
		return errors.Wrapf(err, "unable to parse template %s", template)
	}

	jenkinsScriptName := strings.Split(template, ".")[0]
	configMapName := fmt.Sprintf("%v-%v", instance.Name, jenkinsScriptName)

	configMapData := map[string]string{consts.JenkinsDefaultScriptConfigMapKey: ctx.String()}
	isUpdated, err := platformService.CreateConfigMapWithUpdate(instance, configMapName, configMapData)
	if err != nil {
		return errors.Wrap(err, "unable to create config map")
	}

	_, err = platformService.CreateJenkinsScript(instance.Namespace, configMapName, isUpdated)
	if err != nil {
		return errors.Wrap(err, "unable to create jenkins script")
	}

	return nil
}

func (j JenkinsServiceImpl) createJobProvisions(jobPath string, jc *jenkinsClient.JenkinsClient, instance *jenkinsApi.Jenkins) error {
	jobProvisionsDirectoryPath, err := platformHelper.CreatePathToTemplateDirectory(jobPath)
	if err != nil {
		return err
	}
	configMapName := strings.ReplaceAll(fmt.Sprintf("%v-%v", instance.Name, jobPath), "/", "-")
	configMapKey := consts.JenkinsDefaultScriptConfigMapKey
	env, err := helperController.GetPlatformTypeEnv()
	if err != nil {
		return err
	}
	path := filepath.FromSlash(fmt.Sprintf("%v/%v", jobProvisionsDirectoryPath, env))
	err = j.createScript(instance, configMapName, configMapKey, path)
	return err
}

// IsDeploymentReady check if DC for Jenkins is ready
func (j JenkinsServiceImpl) IsDeploymentReady(instance jenkinsApi.Jenkins) (bool, error) {
	return j.platformService.IsDeploymentReady(instance)
}

func (j JenkinsServiceImpl) createScript(instance *jenkinsApi.Jenkins, configMapName string, configMapKey string, contextPath string) error {
	jenkinsScript, err := j.platformService.CreateJenkinsScript(instance.Namespace, configMapName, false)
	if err != nil {
		return err
	}
	err = j.platformService.CreateConfigMapFromFileOrDir(instance, configMapName, &configMapKey, contextPath, jenkinsScript)
	if err != nil {
		return errors.Wrapf(err, "Couldn't create configs-map %v in namespace %v.", configMapName, instance.Namespace)
	}
	return nil
}

func (j JenkinsServiceImpl) CreateAdminPassword(instance *jenkinsApi.Jenkins) error {
	secretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsAdminPasswordSuffix)
	err := j.createSecret(instance, secretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create Admin password secret")
	}
	if instance.Status.AdminSecretName == "" {
		updatedInstance, err := j.setAdminSecretInStatus(instance, secretName)
		if err != nil {
			return err
		}
		instance = updatedInstance
	}
	return nil
}
