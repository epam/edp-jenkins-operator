package helper

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

const (
	DefaultRequeueTime = 30
)

func NewTrue() *bool {
	value := true
	return &value
}

func NewJenkinsUser(data map[string][]byte, credentialsType string) (JenkinsCredentials, error) {
	out := JenkinsCredentials{}
	switch credentialsType {
	case "ssh":
		params := createSshUserParams(data)
		return JenkinsCredentials{Credentials: params}, nil
	case "password":
		params := createUserWithPassword(data)
		return JenkinsCredentials{Credentials: params}, nil
	default:
		return out, errors.New("Unknown credentials type!")
	}

}

func (user JenkinsCredentials) ToString() (string, error){
	bytes, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	 return string(bytes), nil
}

type JenkinsCredentials struct {
	Credentials JenkinsCredentialsParams `json:"credentials"`
}

type JenkinsCredentialsParams struct {
	Id               string            `json:"id"`
	Scope            string            `json:"scope"`
	Username         string            `json:"username"`
	Password         string            `json:"password,omitempty"`
	Description      string            `json:"description"`
	PrivateKeySource *PrivateKeySource `json:"privateKeySource,omitempty"`
	StaplerClass     StaplerClass      `json:"stapler-class"`
}

type PrivateKeySource struct {
	PrivateKey   string       `json:"privateKey"`
	StaplerClass StaplerClass `json:"stapler-class"`
}

type StaplerClass string

func createUserWithPassword(data map[string][]byte) JenkinsCredentialsParams {
	values := trimNewline(data)
	return JenkinsCredentialsParams{
		Id:           values["username"],
		Scope:        "GLOBAL",
		Username:     values["username"],
		Password:     values["password"],
		Description:  fmt.Sprintf("%s %s", values["first_name"], values["last_name"]),
		StaplerClass: "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
	}
}

func createSshUserParams(data map[string][]byte) JenkinsCredentialsParams {
	values := trimNewline(data)
	return JenkinsCredentialsParams{
		Id:          values["username"],
		Scope:       "GLOBAL",
		Username:    values["username"],
		Password:    string(data["password"]),
		Description: fmt.Sprintf("%s %s", values["first_name"], values["last_name"]),
		PrivateKeySource: &PrivateKeySource{
			PrivateKey:   values["private_key"],
			StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
		},
		StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey",
	}
}

func trimNewline (data map[string][]byte) map[string]string {
	out := map[string]string{}
	for k,v := range data {
		out[k] = strings.TrimSuffix(string(v), "\n")
	}
	return out
}