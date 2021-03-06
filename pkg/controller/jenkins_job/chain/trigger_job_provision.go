package chain

import (
	"context"
	"encoding/json"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type TriggerJobProvision struct {
	next   handler.JenkinsJobHandler
	client client.Client
	ps     platform.PlatformService
	log    logr.Logger
}

func (h TriggerJobProvision) ServeRequest(jj *v1alpha1.JenkinsJob) error {
	h.log.Info("start triggering job provision")

	if err := h.triggerJobProvision(jj); err != nil {
		if err := h.setStatus(jj, consts.StatusFailed, v1alpha1.Error); err != nil {
			return errors.Wrapf(err, "an error has been occurred while updating %v JenkinsJob status", jj.Name)
		}
		return err
	}

	if err := h.setStatus(jj, consts.StatusFinished, v1alpha1.Success); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v JenkinsJob status", jj.Name)
	}
	return nextServeOrNil(h.next, jj)
}

func (h TriggerJobProvision) setStatus(jj *v1alpha1.JenkinsJob, status string, result v1alpha1.Result) error {
	jj.Status = v1alpha1.JenkinsJobStatus{
		Available:                      true,
		LastTimeUpdated:                time.Time{},
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jj.Status.JenkinsJobProvisionBuildNumber,
		Action:                         v1alpha1.TriggerJobProvision,
		Result:                         result,
	}
	return h.updateStatus(jj)
}

func (h TriggerJobProvision) updateStatus(jf *v1alpha1.JenkinsJob) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}

func (h TriggerJobProvision) initGoJenkinsClient(jj v1alpha1.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins job %v", jj.Name)
	}
	h.log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h TriggerJobProvision) triggerJobProvision(jj *v2v1alpha1.JenkinsJob) error {
	h.log.Info("start triggering job provision", "name", jj.Spec.Job.Name)
	jc, err := h.initGoJenkinsClient(*jj)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	var jpc map[string]string
	if err := json.Unmarshal([]byte(jj.Spec.Job.Config), &jpc); err != nil {
		return err
	}

	bn, err := jc.BuildJob(jj.Spec.Job.Name, jpc)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	jj.Status.JenkinsJobProvisionBuildNumber = *bn
	h.log.Info("end triggering build job", "name", jj.Spec.Job.Name)
	return nil
}

func (h TriggerJobProvision) getParamFromJenkinsJobConfig(name, jjConfig string) (*string, error) {
	jobConfig := make(map[string]string)
	err := json.Unmarshal([]byte(jjConfig), &jobConfig)
	if err != nil {
		return nil, err
	}
	var stageName = jobConfig[name]
	return &stageName, nil
}
