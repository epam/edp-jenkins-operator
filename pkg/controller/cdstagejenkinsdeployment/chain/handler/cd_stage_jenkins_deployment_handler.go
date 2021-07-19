package handler

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

type CDStageJenkinsDeploymentHandler interface {
	ServeRequest(jenkinsDeploy *v1alpha1.CDStageJenkinsDeployment) error
}
