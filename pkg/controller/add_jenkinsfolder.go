package controller

import (
	jf "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jf.Add)
}
