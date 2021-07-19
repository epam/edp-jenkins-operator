package chain

import (
	"context"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteCDStageDeploy struct {
	client client.Client
	log    logr.Logger
}

func (h DeleteCDStageDeploy) ServeRequest(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) error {
	log := h.log.WithValues("name", jenkinsDeploy.Spec.Job)
	log.Info("deleting CDStageDeploy")

	if err := h.deleteCDStageDeploy(jenkinsDeploy); err != nil {
		return err
	}

	log.Info("CDStageDeploy has been deleted")
	return nil
}

func (h DeleteCDStageDeploy) deleteCDStageDeploy(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) error {
	s, err := helper.GetCDStageDeploy(h.client, jenkinsDeploy.Labels[consts.CdStageDeployKey], jenkinsDeploy.Namespace)
	if err != nil {
		return err
	}
	return h.client.Delete(context.TODO(), s)
}
