package handler

import jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

type JenkinsJobHandler interface {
	ServeRequest(jj *jenkinsApi.JenkinsJob) error
}
