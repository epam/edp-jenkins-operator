package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
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
		if setStatusErr := h.setStatus(jj, consts.StatusFailed, jenkinsApi.Error); setStatusErr != nil {
			return fmt.Errorf("failed to update %v JenkinsJob status: %w", jj.Name, setStatusErr)
		}

		return err
	}

	if err := h.setStatus(jj, consts.StatusFinished, jenkinsApi.Success); err != nil {
		return fmt.Errorf("failed to update %v JenkinsJob status: %w", jj.Name, err)
	}

	return nextServeOrNil(h.next, jj)
}

func (h TriggerJobProvision) setStatus(jj *jenkinsApi.JenkinsJob, status string, result jenkinsApi.Result) error {
	jj.Status = jenkinsApi.JenkinsJobStatus{
		Available:       true,
		LastTimeUpdated: metav1.NewTime(time.Now()),
		Status:          status,
		Action:          jenkinsApi.TriggerJobProvision,
		Result:          result,
	}

	return h.updateStatus(jj)
}

func (h TriggerJobProvision) updateStatus(jf *jenkinsApi.JenkinsJob) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
	}

	return nil
}

func (h TriggerJobProvision) initGoJenkinsClient(jj *jenkinsApi.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner jenkins for jenkins job %v: %w", jj.Name, err)
	}

	h.log.Info("Jenkins instance has been received", "name", j.Name)

	jClient, err := jenkinsClient.InitGoJenkinsClient(j, h.ps)
	if err != nil {
		return nil, fmt.Errorf("failed to init GoJenkinsClient: %w", err)
	}

	return jClient, nil
}

func (h TriggerJobProvision) triggerJobProvision(jj *jenkinsApi.JenkinsJob) error {
	h.log.Info("start triggering job provision", "name", jj.Spec.Job.Name)

	jc, err := h.initGoJenkinsClient(jj)
	if err != nil {
		return fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	var jpc map[string]string

	if err = json.Unmarshal([]byte(jj.Spec.Job.Config), &jpc); err != nil {
		return fmt.Errorf("failed to unmarshal Jenkins Job Job Config: %w", err)
	}

	bn, err := jc.BuildJob(jj.Spec.Job.Name, jpc)
	if err != nil {
		return fmt.Errorf("failed to trigger job provisioning: %w", err)
	}

	h.log.Info("end triggering build job", "name", jj.Spec.Job.Name, "with BUILD_ID", *bn)

	return nil
}
