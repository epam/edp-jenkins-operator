package chain

import (
	"context"
	"strings"
	"time"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
		return errors.Wrap(err, "an error has been occurred while setting owner reference")
	}

	jc, err := h.initGoJenkinsClient(*jf)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	if err := jc.CreateFolder(jf.Name); err != nil {
		return errors.Wrapf(err, "unable to create %v Jenkins folder", jf.Name)
	}

	if err := h.setStatus(jf, consts.StatusFinished); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", jf.Name)
	}
	log.Info("folder has been created in Jenkins", "name", jf.Name)
	return nextServeOrNil(h.next, jf)
}

func (h PutCDPipelineJenkinsFolder) getCdPipeline(name, namespace string) (*cdPipeApi.CDPipeline, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &cdPipeApi.CDPipeline{}
	if err := h.client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (h PutCDPipelineJenkinsFolder) tryToSetCDPipelineOwnerRef(jf *jenkinsApi.JenkinsFolder) error {
	ow := plutil.GetOwnerReference(consts.CDPipelineKind, jf.GetOwnerReferences())
	if ow != nil {
		log.V(2).Info("cd pipeline owner ref already exists", "jenkins folder", jf.Name)
		return nil
	}

	pn := strings.Replace(jf.Name, "-cd-pipeline", "", -1)
	p, err := h.getCdPipeline(pn, jf.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get CD Pipeline %v from cluster", pn)
	}

	if err := controllerutil.SetControllerReference(p, jf, h.scheme); err != nil {
		return errors.Wrap(err, "couldn't set jenkins owner ref")
	}

	if err := h.client.Update(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", pn)
	}
	return nil
}

func (h PutCDPipelineJenkinsFolder) initGoJenkinsClient(jf jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h PutCDPipelineJenkinsFolder) setStatus(jf *jenkinsApi.JenkinsFolder, status string) error {
	jf.Status = jenkinsApi.JenkinsFolderStatus{
		Available:                      true,
		LastTimeUpdated:                metav1.NewTime(time.Now()),
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return h.updateStatus(jf)
}

func (h PutCDPipelineJenkinsFolder) updateStatus(jf *jenkinsApi.JenkinsFolder) error {
	if err := h.client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}
