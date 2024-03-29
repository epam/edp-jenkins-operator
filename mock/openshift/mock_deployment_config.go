// Code generated by mockery v2.9.4. DO NOT EDIT.

package mock

import (
	context "context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "github.com/openshift/api/apps/v1"

	v1beta1 "k8s.io/api/extensions/v1beta1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// DeploymentConfig is an autogenerated mock type for the DeploymentConfig type
type DeploymentConfig struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, deploymentConfig, opts
func (_m *DeploymentConfig) Create(ctx context.Context, deploymentConfig *v1.DeploymentConfig, opts metav1.CreateOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, deploymentConfig, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, *v1.DeploymentConfig, metav1.CreateOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, deploymentConfig, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.DeploymentConfig, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, deploymentConfig, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *DeploymentConfig) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *DeploymentConfig) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *DeploymentConfig) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, name, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetScale provides a mock function with given fields: ctx, deploymentConfigName, options
func (_m *DeploymentConfig) GetScale(ctx context.Context, deploymentConfigName string, options metav1.GetOptions) (*v1beta1.Scale, error) {
	ret := _m.Called(ctx, deploymentConfigName, options)

	var r0 *v1beta1.Scale
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *v1beta1.Scale); ok {
		r0 = rf(ctx, deploymentConfigName, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1beta1.Scale)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, deploymentConfigName, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Instantiate provides a mock function with given fields: ctx, deploymentConfigName, deploymentRequest, opts
func (_m *DeploymentConfig) Instantiate(ctx context.Context, deploymentConfigName string, deploymentRequest *v1.DeploymentRequest, opts metav1.CreateOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, deploymentConfigName, deploymentRequest, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, string, *v1.DeploymentRequest, metav1.CreateOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, deploymentConfigName, deploymentRequest, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *v1.DeploymentRequest, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, deploymentConfigName, deploymentRequest, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ctx, opts
func (_m *DeploymentConfig) List(ctx context.Context, opts metav1.ListOptions) (*v1.DeploymentConfigList, error) {
	ret := _m.Called(ctx, opts)

	var r0 *v1.DeploymentConfigList
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *v1.DeploymentConfigList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfigList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *DeploymentConfig) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.DeploymentConfig, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Rollback provides a mock function with given fields: ctx, deploymentConfigName, deploymentConfigRollback, opts
func (_m *DeploymentConfig) Rollback(ctx context.Context, deploymentConfigName string, deploymentConfigRollback *v1.DeploymentConfigRollback, opts metav1.CreateOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, deploymentConfigName, deploymentConfigRollback, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, string, *v1.DeploymentConfigRollback, metav1.CreateOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, deploymentConfigName, deploymentConfigRollback, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *v1.DeploymentConfigRollback, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, deploymentConfigName, deploymentConfigRollback, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, deploymentConfig, opts
func (_m *DeploymentConfig) Update(ctx context.Context, deploymentConfig *v1.DeploymentConfig, opts metav1.UpdateOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, deploymentConfig, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, *v1.DeploymentConfig, metav1.UpdateOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, deploymentConfig, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.DeploymentConfig, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, deploymentConfig, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateScale provides a mock function with given fields: ctx, deploymentConfigName, scale, opts
func (_m *DeploymentConfig) UpdateScale(ctx context.Context, deploymentConfigName string, scale *v1beta1.Scale, opts metav1.UpdateOptions) (*v1beta1.Scale, error) {
	ret := _m.Called(ctx, deploymentConfigName, scale, opts)

	var r0 *v1beta1.Scale
	if rf, ok := ret.Get(0).(func(context.Context, string, *v1beta1.Scale, metav1.UpdateOptions) *v1beta1.Scale); ok {
		r0 = rf(ctx, deploymentConfigName, scale, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1beta1.Scale)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *v1beta1.Scale, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, deploymentConfigName, scale, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateStatus provides a mock function with given fields: ctx, deploymentConfig, opts
func (_m *DeploymentConfig) UpdateStatus(ctx context.Context, deploymentConfig *v1.DeploymentConfig, opts metav1.UpdateOptions) (*v1.DeploymentConfig, error) {
	ret := _m.Called(ctx, deploymentConfig, opts)

	var r0 *v1.DeploymentConfig
	if rf, ok := ret.Get(0).(func(context.Context, *v1.DeploymentConfig, metav1.UpdateOptions) *v1.DeploymentConfig); ok {
		r0 = rf(ctx, deploymentConfig, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DeploymentConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.DeploymentConfig, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, deploymentConfig, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *DeploymentConfig) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	var r0 watch.Interface
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) watch.Interface); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(watch.Interface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
