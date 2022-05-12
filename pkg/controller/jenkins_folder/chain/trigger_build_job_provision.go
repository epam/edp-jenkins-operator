package chain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

type TriggerBuildJobProvision struct {
	next   handler.JenkinsFolderHandler
	client client.Client
	ps     platform.PlatformService
}

func (h TriggerBuildJobProvision) ServeRequest(jf *jenkinsApi.JenkinsFolder) error {
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

func (h TriggerBuildJobProvision) setStatus(jf *jenkinsApi.JenkinsFolder, status string) error {
	jf.Status = jenkinsApi.JenkinsFolderStatus{
		Available:                      true,
		LastTimeUpdated:                metav1.NewTime(time.Now()),
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return h.updateStatus(jf)
}

func (h TriggerBuildJobProvision) updateStatus(jf *jenkinsApi.JenkinsFolder) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}

func (h TriggerBuildJobProvision) initGoJenkinsClient(jf jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h TriggerBuildJobProvision) triggerBuildJobProvision(jf *jenkinsApi.JenkinsFolder) error {
	log.V(2).Info("start triggering build job", "name", jf.Spec.Job.Name)
	jc, err := h.initGoJenkinsClient(*jf)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}
	success, err := jc.IsBuildSuccessful(jf.Spec.Job.Name, jf.Status.JenkinsJobProvisionBuildNumber)
	if err != nil {
		return errors.Wrapf(err, "couldn't check build status for job %v", jf.Spec.Job.Name)
	}
	if success {
		log.V(2).Info("last build was successful. triggering of job provision is skipped")
		return nil
	}

	var jpc map[string]string
	err = json.Unmarshal([]byte(jf.Spec.Job.Config), &jpc)
	if err != nil {
		return errors.Wrapf(err, "Cant unmarshal %v", []byte(jf.Spec.Job.Config))
	}

	bn, err := jc.BuildJob(jf.Spec.Job.Name, jpc)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	jf.Status.JenkinsJobProvisionBuildNumber = *bn
	log.Info("end triggering build job", "name", jf.Spec.Job.Name)
	return nil
}
