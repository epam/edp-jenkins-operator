package chain

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	codebase_model "github.com/epmd-edp/codebase-operator/v2/pkg/model"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"time"
)

type TriggerBuildJobProvision struct {
	next handler.JenkinsFolderHandler
	cs   openshift.ClientSet
	ps   platform.PlatformService
}

func (h TriggerBuildJobProvision) ServeRequest(jf *v1alpha1.JenkinsFolder) error {
	log.V(2).Info("start triggering job provision")

	if err := h.triggerBuildJobProvision(jf); err != nil {
		if err := h.setStatus(jf, consts.StatusFailed); err != nil {
			return errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", jf.Name)
		}
		return err
	}

	if err := h.setStatus(jf, consts.StatusFinished); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", jf.Name)
	}
	return nextServeOrNil(h.next, jf)
}

func (h TriggerBuildJobProvision) setStatus(jf *v1alpha1.JenkinsFolder, status string) error {
	jf.Status = v1alpha1.JenkinsFolderStatus{
		Available:                      true,
		LastTimeUpdated:                time.Time{},
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return h.updateStatus(jf)
}

func (h TriggerBuildJobProvision) updateStatus(jf *v1alpha1.JenkinsFolder) error {
	if err := h.cs.Client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.cs.Client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}

func (h TriggerBuildJobProvision) getCodebaseInstanceOwner(jf v2v1alpha1.JenkinsFolder) (*edpv1alpha1.Codebase, error) {
	log.V(2).Info("start getting codebase owner name", "jenkins folder", jf.Name)
	if ow := plutil.GetOwnerReference(consts.CodebaseKind, jf.GetOwnerReferences()); ow != nil {
		log.V(2).Info("trying to fetch codebase owner from reference", "codebase name", ow.Name)
		return h.getCodebaseInstance(ow.Name, jf.Namespace)
	}
	if jf.Spec.CodebaseName != nil {
		log.V(2).Info("trying to fetch codebase owner from spec", "codebase name", jf.Spec.CodebaseName)
		return h.getCodebaseInstance(*jf.Spec.CodebaseName, jf.Namespace)
	}
	return nil, fmt.Errorf("couldn't find codebase owner for jenkins folder %v", jf.Name)
}

func (h TriggerBuildJobProvision) getCodebaseInstance(name, namespace string) (*edpv1alpha1.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &edpv1alpha1.Codebase{}
	if err := h.cs.Client.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return instance, nil
}

func (h TriggerBuildJobProvision) initGoJenkinsClient(jf v1alpha1.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.cs.Client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h TriggerBuildJobProvision) triggerBuildJobProvision(jf *v2v1alpha1.JenkinsFolder) error {
	log.V(2).Info("start triggering build job", "name", jf.Spec.JobName)
	jc, err := h.initGoJenkinsClient(*jf)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	jp := fmt.Sprintf("job-provisions/job/%v", *jf.Spec.JobName)
	success, err := jc.IsBuildSuccessful(jp, jf.Status.JenkinsJobProvisionBuildNumber)
	if err != nil {
		return errors.Wrapf(err, "couldn't check build status for job %v", jp)
	}
	if success {
		log.V(2).Info("last build was successful. triggering of job provision is skipped")
		return nil
	}

	c, err := h.getCodebaseInstanceOwner(*jf)
	if err != nil {
		return errors.Wrapf(err, "an error has been occurred while getting owner codebase for jenkins folder %v", jf.Name)
	}

	gs, err := h.getGitServer(c.Name, c.Spec.GitServer, c.Namespace)
	if err != nil {
		return err
	}
	log.Info("GIT server has been retrieved", "name", gs.Name)

	path := getRepositoryPath(c.Name, string(c.Spec.Strategy), c.Spec.GitUrlPath)
	sshLink := generateSshLink(path, gs)
	jpm := map[string]string{
		"PARAM":                 "true",
		"NAME":                  c.Name,
		"BUILD_TOOL":            strings.ToLower(c.Spec.BuildTool),
		"GIT_SERVER_CR_NAME":    gs.Name,
		"GIT_SERVER_CR_VERSION": "v2",
		"GIT_CREDENTIALS_ID":    gs.NameSshKeySecret,
		"REPOSITORY_PATH":       sshLink,
	}

	bn, err := jc.BuildJob(jp, jpm)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	jf.Status.JenkinsJobProvisionBuildNumber = *bn
	log.Info("end triggering build job", "name", jp)
	return nil
}

func (h TriggerBuildJobProvision) getGitServer(codebaseName, gitServerName, namespace string) (*codebase_model.GitServer, error) {
	gitSec, err := h.getGitServerCR(gitServerName, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while getting Git Server CR for %v codebase", codebaseName)
	}

	gs, err := convertGitServer(*gitSec)
	if err != nil {
		return nil, errors.Wrapf(err, "an error has occurred while converting request Git Server to DTO for %v codebase",
			codebaseName)
	}
	return gs, nil
}

func (h TriggerBuildJobProvision) getGitServerCR(name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.V(2).Info("start getting git server from server", "name", name)
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &edpv1alpha1.GitServer{}
	if err := h.cs.Client.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	log.V(2).Info("end getting git server from server", "name", name)
	return instance, nil
}

func convertGitServer(gs edpv1alpha1.GitServer) (*codebase_model.GitServer, error) {
	log.Info("start converting GitServer", "name", gs.Name)
	if &gs == nil {
		return nil, errors.New("git server object should not be nil")
	}
	return &codebase_model.GitServer{
		Name:             gs.Name,
		GitHost:          gs.Spec.GitHost,
		GitUser:          gs.Spec.GitUser,
		SshPort:          gs.Spec.SshPort,
		NameSshKeySecret: gs.Spec.NameSshKeySecret,
	}, nil
}

func getRepositoryPath(codebaseName, strategy string, gitUrlPath *string) string {
	if strategy == consts.ImportStrategy {
		return *gitUrlPath
	}
	return "/" + codebaseName
}

func generateSshLink(repoPath string, gs *codebase_model.GitServer) string {
	l := fmt.Sprintf("ssh://%v@%v:%v%v", gs.GitUser, gs.GitHost, gs.SshPort, repoPath)
	log.Info("generated SSH link", "link", l)
	return l
}
