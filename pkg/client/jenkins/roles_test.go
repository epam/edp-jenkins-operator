package jenkins

import (
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

func TestIsErrNotFound(t *testing.T) {
	err := errors.Wrap(ErrNotFound("not found"), "error")
	if !IsErrNotFound(err) {
		t.Fatal("error not found is not detected")
	}

	if errors.Cause(err).Error() != "not found" {
		t.Fatal("wrong value of error")
	}

	err = errors.New("fatal")
	if IsErrNotFound(err) {
		t.Fatal("IsErrNotFound is wrong")
	}

}

func TestJenkinsClient_AddRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	httpmock.RegisterResponder("POST", "/role-strategy/strategy/addRole", httpmock.NewStringResponder(200, ""))
	if err := jc.AddRole("rt", "rn", "/*/", []string{"per"}); err != nil {
		t.Fatal(err)
	}
}

func TestJenkinsClient_AssignRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	err := jc.AssignRole("rt", "rn", "s")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "no responder") {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}

func TestJenkinsClient_RemoveRoles(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/removeRoles",
		httpmock.NewStringResponder(200, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	if err := jc.RemoveRoles("rt", []string{"rn"}); err != nil {
		t.Fatal(err)
	}
}

func TestJenkinsClient_UnAssignRole(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/unassignRole",
		httpmock.NewStringResponder(200, ""))

	jc := JenkinsClient{
		resty: restyClient,
	}

	if err := jc.UnAssignRole("rt", "rn", "s"); err != nil {
		t.Fatal(err)
	}
}

func TestJenkinsClient_RoleGet(t *testing.T) {
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	httpmock.RegisterResponder("POST", "/role-strategy/strategy/getRole",
		httpmock.NewStringResponder(200, "{\"foo\": \"bar\"}"))

	jc := JenkinsClient{
		resty: restyClient,
	}

	if _, err := jc.GetRole("rt", "rn"); err != nil {
		t.Fatal(err)
	}
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
	if err == nil {
		t.Fatal("no error returned")
	}

	if !IsErrNotFound(err) {
		t.Fatal("wrong err returned")
	}
}

func TestJenkinsClient_GetRoleFailure_HTTPFailure(t *testing.T) {
	restyClient := resty.New()
	httpmock.DefaultTransport.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	jc := JenkinsClient{
		resty: restyClient,
	}

	_, err := jc.GetRole("rt", "rn")
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "no responder") {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
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
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "status: 500") {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}
