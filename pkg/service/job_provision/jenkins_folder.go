package job_provision

import (
	"context"
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	codebase_model "github.com/epmd-edp/codebase-operator/v2/pkg/model"
	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"

	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("job-provision-service")

type JobProvision struct {
	Client client.Client
}

func (s JobProvision) TriggerBuildJobProvision(jc jenkins.JenkinsClient, c *v1alpha1.Codebase, jf *v2v1alpha1.JenkinsFolder) error {
	log.V(2).Info("start triggering build job", "name", jf.Spec.JobName)
	jp := fmt.Sprintf("job-provisions/job/%v", *jf.Spec.JobName)
	success, err := jc.IsBuildSuccessful(jp, jf.Status.JenkinsJobProvisionBuildNumber)
	if err != nil {
		return errors.Wrapf(err, "couldn't check build status for job %v", jp)
	}
	if success {
		log.V(2).Info("last build was successful. triggering of job provision is skipped")
		return nil
	}

	gs, err := s.getGitServer(c.Name, c.Spec.GitServer, c.Namespace)
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
	log.V(2).Info("end triggering build job", "name", jp)
	return nil
}

func (s JobProvision) getGitServer(codebaseName, gitServerName, namespace string) (*codebase_model.GitServer, error) {
	gitSec, err := s.getGitServerCR(gitServerName, namespace)
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

func (s JobProvision) getGitServerCR(name, namespace string) (*edpv1alpha1.GitServer, error) {
	log.V(2).Info("start getting git server from server", "name", name)
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &edpv1alpha1.GitServer{}
	if err := s.Client.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	log.V(2).Info("end getting git server from server", "name", name)
	return instance, nil
}

func convertGitServer(gs v1alpha1.GitServer) (*codebase_model.GitServer, error) {
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
