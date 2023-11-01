package jenkins

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/resty.v1"
)

const (
	crTypeKey     = "type"
	crRoleNameKey = "roleName"
)

type Role struct {
	PermissionIDs map[string]bool `json:"permissionIds"`
	Pattern       string          `json:"pattern"`
	SIDs          []string        `json:"sids"`
}

var ErrNotFound = errors.New("role is not found")

func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// AddRole add role to jenkins
// roleType - type of role, available options: globalRoles, projectRoles, nodeRoles.
func (jc JenkinsClient) AddRole(roleType, name, pattern string, permissions []string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		crTypeKey:       roleType,
		crRoleNameKey:   name,
		"pattern":       pattern,
		"permissionIds": strings.Join(permissions, ","),
	}).Post("/role-strategy/strategy/addRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) RemoveRoles(roleType string, roleNames []string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		crTypeKey:   roleType,
		"roleNames": strings.Join(roleNames, ","),
	}).Post("/role-strategy/strategy/removeRoles")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) AssignRole(roleType, roleName, subject string) error {
	if _, err := jc.GetRole(roleType, roleName); err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	rsp, err := jc.resty.R().SetFormData(map[string]string{
		crTypeKey:     roleType,
		crRoleNameKey: roleName,
		"sid":         subject,
	}).Post("/role-strategy/strategy/assignRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) UnAssignRole(roleType, roleName, subject string) error {
	rsp, err := jc.resty.R().SetFormData(map[string]string{
		crTypeKey:     roleType,
		crRoleNameKey: roleName,
		"sid":         subject,
	}).Post("/role-strategy/strategy/unassignRole")

	return parseRestyResponse(rsp, err)
}

func (jc JenkinsClient) GetRole(roleType, roleName string) (*Role, error) {
	var r Role

	rsp, err := jc.resty.R().SetFormData(map[string]string{
		crTypeKey:     roleType,
		crRoleNameKey: roleName,
	}).SetResult(&r).Post("/role-strategy/strategy/getRole")

	if err = parseRestyResponse(rsp, err); err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if rsp.String() == "{}" {
		return nil, ErrNotFound
	}

	return &r, nil
}

func parseRestyResponse(rsp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("failed to perform post request: %w", err)
	}

	if rsp.IsError() {
		return fmt.Errorf("failed to run request: status: %s, body: %s", rsp.Status(), rsp.String())
	}

	return nil
}
