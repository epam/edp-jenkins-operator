package jenkins

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"gopkg.in/resty.v1"
)

func TestIsErrNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want require.BoolAssertionFunc
	}{
		{
			name: "should be IsErrNotFound",
			err:  fmt.Errorf("error: %w", ErrNotFound),
			want: require.True,
		},
		{
			name: "should not be IsErrNotFound",
			err:  errors.New("other err"),
			want: require.False,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.want(t, IsErrNotFound(tt.err))
		})
	}
}

func TestJenkinsClient_AddRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder(http.MethodPost, "/role-strategy/strategy/addRole",
		httpmock.NewStringResponder(200, ""))

	require.NoError(t, jc.AddRole("rt", "rn", "/*/", []string{"per"}))
}

func TestJenkinsClient_AssignRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	err := jc.AssignRole("rt", "rn", "s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "role-strategy/strategy/getRole\": no responder found")

	httpmock.RegisterResponder("POST", "/role-strategy/strategy/getRole",
		httpmock.NewJsonResponderOrPanic(200, Role{}))
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/assignRole",
		httpmock.NewStringResponder(200, ""))

	require.NoError(t, jc.AssignRole("rt", "rn", "s"))
}

func TestJenkinsClient_RemoveRoles(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/removeRoles",
		httpmock.NewStringResponder(200, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	require.NoError(t, jc.RemoveRoles("rt", []string{"rn"}))
}

func TestJenkinsClient_UnAssignRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/unassignRole",
		httpmock.NewStringResponder(200, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	require.NoError(t, jc.UnAssignRole("rt", "rn", "s"))
}

func TestJenkinsClient_RoleGet(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/getRole",
		httpmock.NewStringResponder(200, "{\"foo\": \"bar\"}"))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetRole("rt", "rn")
	require.NoError(t, err)
}

func TestJenkinsClient_GetRoleFailure_NoRoles(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/getRole",
		httpmock.NewStringResponder(200, "{}"))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetRole("rt", "rn")
	require.Error(t, err)

	require.True(t, IsErrNotFound(err))
}

func TestJenkinsClient_GetRoleFailure_HTTPFailure(t *testing.T) {
	restyClient := resty.New()

	httpmock.DefaultTransport.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetRole("rt", "rn")
	require.Error(t, err)

	require.Contains(t, err.Error(), "no responder")
}

func TestJenkinsClient_GetRoleFailure_500(t *testing.T) {
	restyClient := resty.New()

	httpmock.DefaultTransport.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/getRole",
		httpmock.NewStringResponder(500, "{}"))

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetRole("rt", "rn")
	require.Error(t, err)

	require.Contains(t, err.Error(), "status: 500")
}
