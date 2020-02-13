package chain

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bndr/gojenkins"
	pipev1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebasev1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	handler2 "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"text/template"
)

type PutJenkinsPipeline struct {
	next handler2.JenkinsJobHandler
	cs   openshift.ClientSet
	ps   platform.PlatformService
}

func (h PutJenkinsPipeline) ServeRequest(jj *v1alpha1.JenkinsJob) error {
	log.V(2).Info("start creating Jenkins CD Pipeline")
	if err := setIntermediateStatus(h.cs.Client, jj, v1alpha1.CreateJenkinsPipeline); err != nil {
		return err
	}
	if err := h.tryToCreateJob(jj); err != nil {
		if err := setFailStatus(h.cs.Client, jj, v1alpha1.CreateJenkinsPipeline, err.Error()); err != nil {
			return err
		}
		return err
	}
	if err := setFinishStatus(h.cs.Client, jj, v1alpha1.CreateJenkinsPipeline); err != nil {
		return err
	}
	log.V(2).Info("end creating Jenkins CD Pipeline")
	return nextServeOrNil(h.next, jj)
}

func (h PutJenkinsPipeline) tryToCreateJob(jj *v1alpha1.JenkinsJob) error {
	jc, err := h.initGoJenkinsClient(jj)
	if err != nil {
		return err
	}

	jp := h.getJobName(jj)
	job, err := h.getJob(jc, jp)
	if err != nil {
		return err
	}
	if job != nil {
		log.V(2).Info("job already exists. skip creating", "name", jp)
		return nil
	}

	s, err := plutil.GetStageInstanceOwner(h.cs.Client, *jj)
	if err != nil {
		return err
	}

	json, err := h.ps.CreateStageJSON(*s)
	if err != nil {
		return err
	}

	conf, err := h.createStageConfig(s, json, jj.Spec.Job.Config)
	if err != nil {
		return err
	}

	if err := h.createJob(jc, conf, jj); err != nil {
		return errors.Wrap(err, "couldn't create jenkins job")
	}

	log.Info("job has been created", "name", jp)
	return nil
}

func (h PutJenkinsPipeline) getJobName(jj *v1alpha1.JenkinsJob) string {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		return fmt.Sprintf("%v-cd-pipeline/job/%v", *jj.Spec.JenkinsFolder, jj.Spec.Job.Name)
	}
	return jj.Spec.Job.Name
}

func (h PutJenkinsPipeline) createJob(jc *jenkinsClient.JenkinsClient, conf *string, jj *v1alpha1.JenkinsJob) error {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		pfn := fmt.Sprintf("%v-%v", *jj.Spec.JenkinsFolder, "cd-pipeline")
		_, err := jc.GoJenkins.CreateJobInFolder(*conf, jj.Spec.Job.Name, pfn)
		if err != nil {
			return err
		}
		log.V(2).Info("job has been created",
			"name", fmt.Sprintf("%v/%v", pfn, jj.Spec.Job.Name))
		return nil
	}
	if _, err := jc.GoJenkins.CreateJob(*conf, jj.Spec.Job.Name); err != nil {
		return err
	}
	log.V(2).Info("job has been created", "name", jj.Spec.Job.Name)
	return nil
}

func (h PutJenkinsPipeline) getJob(jc *jenkinsClient.JenkinsClient, jp string) (*gojenkins.Job, error) {
	job, err := jc.GoJenkins.GetJob(jp)
	if err != nil {
		if err.Error() == "404" {
			log.V(2).Info("job doesn't exist. start creating", "name", jp)
			return nil, nil
		}
		return nil, err
	}
	return job, nil
}

func (h PutJenkinsPipeline) initGoJenkinsClient(jj *v1alpha1.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.cs.Client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins job %v", jj.Name)
	}
	log.V(2).Info("Jenkins instance has been created", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h PutJenkinsPipeline) createStageConfig(s *pipev1alpha1.Stage, ps, conf string) (*string, error) {
	pipeSrc := map[string]interface{}{
		"type":    "default",
		"library": map[string]string{},
	}

	if s.Spec.Source.Type == consts.LibraryCodebase {
		h.setPipeSrcParams(s, pipeSrc)
	}

	tmpl, err := template.New("cd-pipeline.tmpl").Parse(conf)
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"name":               s.Spec.Name,
		"gitServerCrVersion": "v2",
		"pipelineStages":     ps,
		"source":             pipeSrc,
	}
	var cdPipelineBuffer bytes.Buffer
	if err := tmpl.Execute(&cdPipelineBuffer, params); err != nil {
		return nil, err
	}
	pipeConf := cdPipelineBuffer.String()
	return &pipeConf, nil
}

func (h PutJenkinsPipeline) setPipeSrcParams(stage *pipev1alpha1.Stage, pipeSrc map[string]interface{}) {
	cb, err := h.getLibraryParams(stage.Spec.Source.Library.Name, stage.Namespace)
	if err != nil {
		log.Error(err, "couldn't retrieve parameters for pipeline's library, default source type will be used",
			"Library name", stage.Spec.Source.Library.Name)
	}
	gs, err := h.getGitServerParams(cb.Spec.GitServer, stage.Namespace)
	if err != nil {
		log.Error(err, "couldn't retrieve parameters for git server, default source type will be used",
			"Git server", cb.Spec.GitServer)
	} else {
		pipeSrc["type"] = "library"
		pipeSrc["library"] = map[string]string{
			"url": fmt.Sprintf("ssh://%v@%v:%v%v", gs.Spec.GitUser, gs.Spec.GitHost, gs.Spec.SshPort,
				getPathToRepository(string(cb.Spec.Strategy), stage.Spec.Source.Library.Name, cb.Spec.GitUrlPath)),
			"credentials": gs.Spec.NameSshKeySecret,
			"branch":      stage.Spec.Source.Library.Branch,
		}
	}
}

func (h PutJenkinsPipeline) getLibraryParams(name, ns string) (*codebasev1alpha1.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebasev1alpha1.Codebase{}
	if err := h.cs.Client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (h PutJenkinsPipeline) getGitServerParams(name string, ns string) (*codebasev1alpha1.GitServer, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebasev1alpha1.GitServer{}
	if err := h.cs.Client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func getPathToRepository(strategy, name string, url *string) string {
	if strategy == consts.ImportStrategy {
		return *url
	}
	return "/" + name
}
