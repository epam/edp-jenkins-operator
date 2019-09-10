package controller

import (
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsserviceaccount"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jenkinsserviceaccount.Add)
}
