package chain

import (
	"context"
	pipe_v1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

type PutCDPipelineJenkinsFolder struct {
	next   handler.JenkinsFolderHandler
	cs     openshift.ClientSet
	ps     platform.PlatformService
	scheme *runtime.Scheme
}

func (h PutCDPipelineJenkinsFolder) ServeRequest(jf *v1alpha1.JenkinsFolder) error {
	log.V(2).Info("start creating cd pipeline folder in Jenkins", "name", jf.Name)

	if err := h.tryToSetCDPipelineOwnerRef(jf); err != nil {
		return errors.Wrap(err, "an error has been occurred while setting owner reference")
	}

	jc, err := h.initGoJenkinsClient(*jf)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	if err := jc.CreateFolder(jf.Name); err != nil {
		return err
	}

	if err := h.setStatus(jf, consts.StatusFinished); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", jf.Name)
	}
	log.Info("folder has been created in Jenkins", "name", jf.Name)
	return nextServeOrNil(h.next, jf)
}

func (h PutCDPipelineJenkinsFolder) getCdPipeline(name, namespace string) (*pipe_v1alpha1.CDPipeline, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &pipe_v1alpha1.CDPipeline{}
	if err := h.cs.Client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (h PutCDPipelineJenkinsFolder) tryToSetCDPipelineOwnerRef(jf *v1alpha1.JenkinsFolder) error {
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

	if err := h.cs.Client.Update(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", pn)
	}
	return nil
}

func (h PutCDPipelineJenkinsFolder) initGoJenkinsClient(jf v1alpha1.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(h.cs.Client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, h.ps)
}

func (h PutCDPipelineJenkinsFolder) setStatus(jf *v1alpha1.JenkinsFolder, status string) error {
	jf.Status = v1alpha1.JenkinsFolderStatus{
		Available:                      true,
		LastTimeUpdated:                time.Time{},
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return h.updateStatus(jf)
}

func (h PutCDPipelineJenkinsFolder) updateStatus(jf *v1alpha1.JenkinsFolder) error {
	if err := h.cs.Client.Status().Update(context.TODO(), jf); err != nil {
		if err := h.cs.Client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}
