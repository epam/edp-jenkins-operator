// Code generated by mockery v2.9.4. DO NOT EDIT.

package mock

import (
	v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	mock "github.com/stretchr/testify/mock"
)

// JenkinsService is an autogenerated mock type for the JenkinsService type
type JenkinsService struct {
	mock.Mock
}

// Configure provides a mock function with given fields: instance
func (_m *JenkinsService) Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	ret := _m.Called()

	var r0 *v1alpha1.Jenkins
	if rf, ok := ret.Get(0).(func(v1alpha1.Jenkins) *v1alpha1.Jenkins); ok {
		r0 = rf(instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Jenkins)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(v1alpha1.Jenkins) bool); ok {
		r1 = rf(instance)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(v1alpha1.Jenkins) error); ok {
		r2 = rf(instance)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateAdminPassword provides a mock function with given fields: instance
func (_m *JenkinsService) CreateAdminPassword(instance v1alpha1.Jenkins) error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func(v1alpha1.Jenkins) error); ok {
		r0 = rf(instance)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ExposeConfiguration provides a mock function with given fields: instance
func (_m *JenkinsService) ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	ret := _m.Called()

	var r0 *v1alpha1.Jenkins
	if rf, ok := ret.Get(0).(func(v1alpha1.Jenkins) *v1alpha1.Jenkins); ok {
		r0 = rf(instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Jenkins)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(v1alpha1.Jenkins) bool); ok {
		r1 = rf(instance)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(v1alpha1.Jenkins) error); ok {
		r2 = rf(instance)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Integration provides a mock function with given fields: instance
func (_m *JenkinsService) Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	ret := _m.Called()

	var r0 *v1alpha1.Jenkins
	if rf, ok := ret.Get(0).(func(v1alpha1.Jenkins) *v1alpha1.Jenkins); ok {
		r0 = rf(instance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Jenkins)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(v1alpha1.Jenkins) bool); ok {
		r1 = rf(instance)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(v1alpha1.Jenkins) error); ok {
		r2 = rf(instance)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// IsDeploymentReady provides a mock function with given fields: instance
func (_m *JenkinsService) IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error) {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func(v1alpha1.Jenkins) bool); ok {
		r0 = rf(instance)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(v1alpha1.Jenkins) error); ok {
		r1 = rf(instance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}