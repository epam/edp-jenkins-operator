package chain

import (
	"context"
	"encoding/json"
	"github.com/epam/edp-codebase-operator/v2/pkg/openshift"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
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

func (h TriggerBuildJobProvision) initGoJenkinsClient(jf v1alpha1.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.cs.Client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h TriggerBuildJobProvision) triggerBuildJobProvision(jf *v2v1alpha1.JenkinsFolder) error {
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

	bn, err := jc.BuildJob(jf.Spec.Job.Name, jpc)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while triggering job provisioning")
	}
	jf.Status.JenkinsJobProvisionBuildNumber = *bn
	log.Info("end triggering build job", "name", jf.Spec.Job.Name)
	return nil
}
