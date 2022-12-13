package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

type PutCDPipelineJenkinsFolder struct {
	next   handler.JenkinsFolderHandler
	client client.Client
	ps     platform.PlatformService
	scheme *runtime.Scheme
}

func (h PutCDPipelineJenkinsFolder) ServeRequest(jf *jenkinsApi.JenkinsFolder) error {
	log.V(2).Info("start creating cd pipeline folder in Jenkins", "name", jf.Name)

	if err := h.tryToSetCDPipelineOwnerRef(jf); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	jc, err := h.initGoJenkinsClient(jf)
	if err != nil {
		return fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	if err := jc.CreateFolder(jf.Name); err != nil {
		return fmt.Errorf("failed to create %v Jenkins folder: %w", jf.Name, err)
	}

	if err := h.setStatus(jf, consts.StatusFinished); err != nil {
		return fmt.Errorf("failed to update JobFolder \"%s\" status: %w", jf.Name, err)
	}

	log.Info("folder has been created in Jenkins", "name", jf.Name)

	return nextServeOrNil(h.next, jf)
}

func (h PutCDPipelineJenkinsFolder) getCdPipeline(name, namespace string) (*cdPipeApi.CDPipeline, error) {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	cdPipeline := &cdPipeApi.CDPipeline{}

	if err := h.client.Get(context.TODO(), namespacedName, cdPipeline); err != nil {
		return nil, fmt.Errorf("failed to get CDPipeline: %w", err)
	}

	return cdPipeline, nil
}

func (h PutCDPipelineJenkinsFolder) tryToSetCDPipelineOwnerRef(jf *jenkinsApi.JenkinsFolder) error {
	ow := plutil.GetOwnerReference(consts.CDPipelineKind, jf.GetOwnerReferences())
	if ow != nil {
		log.V(2).Info("cd pipeline owner ref already exists", "jenkins folder", jf.Name)

		return nil
	}

	pn := strings.ReplaceAll(jf.Name, "-cd-pipeline", "")

	p, err := h.getCdPipeline(pn, jf.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get CD Pipeline %v from cluster: %w", pn, err)
	}

	if err = controllerutil.SetControllerReference(p, jf, h.scheme); err != nil {
		return fmt.Errorf("failed to set jenkins owner ref: %w", err)
	}

	if err = h.client.Update(context.TODO(), jf); err != nil {
		return fmt.Errorf("failed to update jenkins job %v: %w", pn, err)
	}

	return nil
}

func (h PutCDPipelineJenkinsFolder) initGoJenkinsClient(jf *jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner jenkins for jenkins folder %v: %w", jf.Name, err)
	}

	log.Info("Jenkins instance has been received", "name", j.Name)

	jenkinsCl, err := jenkinsClient.InitGoJenkinsClient(j, h.ps)
	if err != nil {
		return nil, fmt.Errorf("failed to init Jenkins Client: %w", err)
	}

	return jenkinsCl, nil
}

func (h PutCDPipelineJenkinsFolder) setStatus(jf *jenkinsApi.JenkinsFolder, status string) error {
	jf.Status = jenkinsApi.JenkinsFolderStatus{
		Available:       true,
		LastTimeUpdated: metav1.NewTime(time.Now()),
		Status:          status,
	}

	return h.updateStatus(jf)
}

func (h PutCDPipelineJenkinsFolder) updateStatus(jf *jenkinsApi.JenkinsFolder) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return fmt.Errorf("failed to update JenkinsFolder: %w", err)
		}
	}

	return nil
}
