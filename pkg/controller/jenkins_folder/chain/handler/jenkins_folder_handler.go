package handler

import "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"

type JenkinsFolderHandler interface {
	ServeRequest(jf *v1alpha1.JenkinsFolder) error
}
