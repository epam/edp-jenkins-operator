package handler

import "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"

type JenkinsJobHandler interface {
	ServeRequest(jj *v1alpha1.JenkinsJob) error
}
