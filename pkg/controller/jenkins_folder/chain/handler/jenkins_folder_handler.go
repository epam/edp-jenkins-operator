package handler

import jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

type JenkinsFolderHandler interface {
	ServeRequest(jf *jenkinsApi.JenkinsFolder) error
}
