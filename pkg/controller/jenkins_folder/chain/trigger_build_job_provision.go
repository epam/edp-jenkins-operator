package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
		if setStatusErr := h.setStatus(jf, consts.StatusFailed); setStatusErr != nil {
			return fmt.Errorf("failed to update %v JobFolder status: %w", jf.Name, setStatusErr)
		}

		return fmt.Errorf("failed to trigger job provision build: %w", err)
	}

	if err := h.setStatus(jf, consts.StatusFinished); err != nil {
		return fmt.Errorf("failed to update %v JobFolder status: %w", jf.Name, err)
	}

	return nextServeOrNil(h.next, jf)
}

func (h TriggerBuildJobProvision) setStatus(jf *jenkinsApi.JenkinsFolder, status string) error {
	jf.Status = jenkinsApi.JenkinsFolderStatus{
		Available:       true,
		LastTimeUpdated: metav1.NewTime(time.Now()),
		Status:          status,
	}

	return h.updateStatus(jf)
}

func (h TriggerBuildJobProvision) updateStatus(jf *jenkinsApi.JenkinsFolder) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return fmt.Errorf("failed to update client: %w", err)
		}
	}

	return nil
}

func (h TriggerBuildJobProvision) initGoJenkinsClient(jf *jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner jenkins for jenkins folder %v: %w", jf.Name, err)
	}

	log.Info("Jenkins instance has been received", "name", j.Name)

	jClient, err := jenkinsClient.InitGoJenkinsClient(j, h.ps)
	if err != nil {
		return nil, fmt.Errorf("failed to init GoJenkinsClient: %w", err)
	}

	return jClient, nil
}

func (h TriggerBuildJobProvision) triggerBuildJobProvision(jf *jenkinsApi.JenkinsFolder) error {
	if jf.Spec.Job == nil {
		return errors.New("failed to start to build - job field is empty in spec")
	}

	log.V(2).Info("start triggering build job", "name", jf.Spec.Job.Name)

	jc, err := h.initGoJenkinsClient(jf)
	if err != nil {
		return fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	var jpc map[string]string

	if err = json.Unmarshal([]byte(jf.Spec.Job.Config), &jpc); err != nil {
		return fmt.Errorf("failed to Unmarshal %v: %w", []byte(jf.Spec.Job.Config), err)
	}

	bn, err := jc.BuildJob(jf.Spec.Job.Name, jpc)
	if err != nil {
		return fmt.Errorf("failed to build job provisioning: %w", err)
	}

	log.Info("end triggering build job", "name", jf.Spec.Job.Name, "with BUILD_ID", *bn)

	return nil
}
