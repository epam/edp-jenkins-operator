package controller

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins"
	jf "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder"
	jj "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkinsscript"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkinsserviceaccount"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, jenkins.Add, jf.Add, jj.Add,
		jenkinsscript.Add, jenkinsserviceaccount.Add, cdstagejenkinsdeployment.Add)
}
