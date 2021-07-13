package jenkins

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type Role struct {
	PermissionIDs map[string]bool `json:"permissionIds"`
	Pattern       string          `json:"pattern"`
	SIDs          []string        `json:"sids"`
}

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return string(e)
}

func IsErrNotFound(err error) bool {
	switch errors.Cause(err).(type) {
	case ErrNotFound:
		return true
	}

	return false
}

// AddRole add role to jenkins
// roleType - type of role, available options: globalRoles, projectRoles, nodeRoles
func (jc JenkinsClient) AddRole(roleType, name, pattern string, permissions []string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		"type":          roleType,
		"roleName":      name,
		"pattern":       pattern,
		"permissionIds": strings.Join(permissions, ","),
		"overwrite":     "true",
	}).Post("/role-strategy/strategy/addRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) RemoveRoles(roleType string, roleNames []string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		"type":      roleType,
		"roleNames": strings.Join(roleNames, ","),
	}).Post("/role-strategy/strategy/removeRoles")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) AssignRole(roleType, roleName, subject string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		"type":     roleType,
		"roleName": roleName,
		"sid":      subject,
	}).Post("/role-strategy/strategy/assignRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) UnAssignRole(roleType, roleName, subject string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		"type":     roleType,
		"roleName": roleName,
		"sid":      subject,
	}).Post("/role-strategy/strategy/unassignRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) GetRole(roleType, roleName string) (*Role, error) {
	var r Role

	rsp, err := jc.resty.R().SetFormData(map[string]string{
		"type":     roleType,
		"roleName": roleName,
	}).SetResult(&r).Post("/role-strategy/strategy/getRole")

	if err := parseRestyResponse(rsp, err); err != nil {
		return nil, errors.Wrap(err, "unable to get role")
	}

	if rsp.String() == "{}" {
		return nil, ErrNotFound("role is not found")
	}

	return &r, nil
}

func parseRestyResponse(rsp *resty.Response, err error) error {
	if err != nil {
		return errors.Wrap(err, "error during post request")
	}

	if rsp.IsError() {
		return errors.Errorf("status: %s, body: %s", rsp.Status(), rsp.String())
	}

	return nil
}
