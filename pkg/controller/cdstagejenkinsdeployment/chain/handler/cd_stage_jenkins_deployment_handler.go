package handler

import (
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

type CDStageJenkinsDeploymentHandler interface {
	ServeRequest(jenkinsDeploy *jenkinsApi.CDStageJenkinsDeployment) error
}
