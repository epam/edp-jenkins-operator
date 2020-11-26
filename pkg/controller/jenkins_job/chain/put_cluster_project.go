package chain

import (
	"context"
	"fmt"
	pipev1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	handler2 "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PutClusterProject struct {
	next handler2.JenkinsJobHandler
	cs   openshift.ClientSet
	ps   platform.PlatformService
}

func (h PutClusterProject) ServeRequest(jj *v1alpha1.JenkinsJob) error {
	log.V(2).Info("start creating project in cluster")
	if err := setIntermediateStatus(h.cs.Client, jj, v1alpha1.PlatformProjectCreation); err != nil {
		return err
	}
	if err := h.tryToCreateProject(jj); err != nil {
		if err := setFailStatus(h.cs.Client, jj, v1alpha1.PlatformProjectCreation, err.Error()); err != nil {
			return err
		}
		return err
	}
	log.V(2).Info("end creating project in cluster")
	return nextServeOrNil(h.next, jj)
}

func (h PutClusterProject) tryToCreateProject(jj *v1alpha1.JenkinsJob) error {
	d, err := h.ps.GetConfigMapData(jj.Namespace, "edp-config")
	if err != nil {
		return err
	}
	s, err := plutil.GetStageInstanceOwner(h.cs.Client, *jj)
	if err != nil {
		return err
	}
	pn := fmt.Sprintf("%v-%v", d["edp_name"], s.Name)
	if err := h.createProject(*s, pn); err != nil {
		return err
	}
	log.Info("project has been created", "name", pn)
	return nil
}

func (h PutClusterProject) createProject(s pipev1alpha1.Stage, name string) error {
	if err := h.ps.CreateProject(name); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.V(2).Info("project already exists. skip creating...", "name", name)
			return nil
		}
		return err
	}
	return nil
}

func setIntermediateStatus(c client.Client, jj *v1alpha1.JenkinsJob, action v1alpha1.ActionType) error {
	jj.Status = v1alpha1.JenkinsJobStatus{
		Status:          util.StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Action:          action,
		Result:          "success",
		Username:        "system",
		Value:           "inactive",
	}
	return updateStatus(c, jj)
}

func setFailStatus(c client.Client, jj *v1alpha1.JenkinsJob, action v1alpha1.ActionType, msg string) error {
	jj.Status = v1alpha1.JenkinsJobStatus{
		Status:          util.StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          v1alpha1.Error,
		DetailedMessage: msg,
		Value:           "failed",
	}
	return updateStatus(c, jj)

}

func setFinishStatus(c client.Client, jj *v1alpha1.JenkinsJob, action v1alpha1.ActionType) error {
	jj.Status = v1alpha1.JenkinsJobStatus{
		Status:          util.StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Action:          action,
		Result:          v1alpha1.Success,
		Username:        "system",
		Value:           "active",
	}
	return updateStatus(c, jj)
}

func updateStatus(c client.Client, jj *v1alpha1.JenkinsJob) error {
	if err := c.Status().Update(context.TODO(), jj); err != nil {
		if err := c.Update(context.TODO(), jj); err != nil {
			return errors.Wrap(err, "couldn't update jenkins job status")
		}
	}
	log.Info("JenkinsJob status has been updated", "name", jj.Name)
	return nil
}
