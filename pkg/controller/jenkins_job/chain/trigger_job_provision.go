package chain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

type TriggerJobProvision struct {
	next   handler.JenkinsJobHandler
	client client.Client
	ps     platform.PlatformService
	log    logr.Logger
}

func (h TriggerJobProvision) ServeRequest(jj *jenkinsApi.JenkinsJob) error {
	h.log.Info("start triggering job provision")

	if err := h.triggerJobProvision(jj); err != nil {
		if err := h.setStatus(jj, consts.StatusFailed, jenkinsApi.Error); err != nil {
			return errors.Wrapf(err, "an error has been occurred while updating %v JenkinsJob status", jj.Name)
		}
		return err
	}

	if err := h.setStatus(jj, consts.StatusFinished, jenkinsApi.Success); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v JenkinsJob status", jj.Name)
	}
	return nextServeOrNil(h.next, jj)
}

func (h TriggerJobProvision) setStatus(jj *jenkinsApi.JenkinsJob, status string, result jenkinsApi.Result) error {
	jj.Status = jenkinsApi.JenkinsJobStatus{
		Available:                      true,
		LastTimeUpdated:                metav1.NewTime(time.Now()),
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jj.Status.JenkinsJobProvisionBuildNumber,
		Action:                         jenkinsApi.TriggerJobProvision,
		Result:                         result,
	}
	return h.updateStatus(jj)
}

func (h TriggerJobProvision) updateStatus(jf *jenkinsApi.JenkinsJob) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}

func (h TriggerJobProvision) initGoJenkinsClient(jj jenkinsApi.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins job %v", jj.Name)
	}
	h.log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h TriggerJobProvision) triggerJobProvision(jj *jenkinsApi.JenkinsJob) error {
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
