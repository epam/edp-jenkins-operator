package helper

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

const (
	DefaultRequeueTime        = 30
	GlobalScope        string = "GLOBAL"
	SSHUserType         string = "ssh"
	PasswordUserType   string = "password"
	TokenUserType      string = "token"
)

func NewTrue() *bool {
	value := true
	return &value
}

func NewJenkinsUser(data map[string][]byte, credentialsType string) (JenkinsCredentials, error) {
	out := JenkinsCredentials{}
	switch credentialsType {
	case SSHUserType:
		params := createSshUserParams(data)
		return JenkinsCredentials{Credentials: params}, nil
	case PasswordUserType:
		params := createUserWithPassword(data)
		return JenkinsCredentials{Credentials: params}, nil
	case TokenUserType:
		params := createStringCredentials(data)
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
	Username         *string           `json:"username,omitempty"`
	Password         *string           `json:"password,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Secret           *string           `json:"secret,omitempty"`
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
	username := values["username"]
	password := values["password"]
	description := fmt.Sprintf("%s %s", values["first_name"], values["last_name"])
	return JenkinsCredentialsParams{
		Id:           username,
		Scope:        GlobalScope,
		Username:     &username,
		Password:     &password,
		Description:  &description,
		StaplerClass: "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
	}
}

func createSshUserParams(data map[string][]byte) JenkinsCredentialsParams {
	values := trimNewline(data)
	username := values["username"]
	password := values["password"]
	description := fmt.Sprintf("%s %s", values["first_name"], values["last_name"])
	return JenkinsCredentialsParams{
		Id:          username,
		Scope:       GlobalScope,
		Username:    &username,
		Password:    &password,
		Description: &description,
		PrivateKeySource: &PrivateKeySource{
			PrivateKey:   values["id_rsa"],
			StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
		},
		StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey",
	}
}

func createStringCredentials(data map[string][]byte) JenkinsCredentialsParams {
	values := trimNewline(data)
	secret := values["secret"]
	description := values["username"]
	return JenkinsCredentialsParams{
		Id:           values["username"],
		Scope:        GlobalScope,
		Secret:       &secret,
		Description:  &description,
		StaplerClass: "org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl",
	}
}

func trimNewline(data map[string][]byte) map[string]string {
	out := map[string]string{}
	for k,v := range data {
		out[k] = strings.TrimSuffix(string(v), "\n")
	}
	return out
}