package mock

import (
	"context"
	mock "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	mock.Client
}

func (c *Client) Update(ctx context.Context, obj client.Object, options ...client.UpdateOption) error {
	called := c.Called()
	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.Update(ctx, obj, options...)
	}

	return called.Error(0)
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	called := c.Called(list)
	parent, ok := called.Get(0).(client.Client)
	if ok {
		return parent.List(ctx, list, opts...)
	}

	return called.Error(0)
}
