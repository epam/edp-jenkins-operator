package controller

import (
	"jenkins-operator/pkg/controller/jenkinsscript"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jenkinsscript.Add)
}
