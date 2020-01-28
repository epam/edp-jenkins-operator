package controller

import (
	jj "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jj.Add)
}
