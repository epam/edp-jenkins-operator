package helper

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"regexp"
	"strings"
)

const (
	DefaultRequeueTime        = 30
	GlobalScope        string = "GLOBAL"
	SSHUserType        string = "ssh"
	PasswordUserType   string = "password"
	TokenUserType      string = "token"
	platformType       string = "PLATFORM_TYPE"
)

func NewTrue() *bool {
	value := true
	return &value
}

func generateCredentialsMap(rawData map[string][]byte) (map[string]string, error) {
	data := trimNewline(rawData)
	if _, ok := data["id"]; ok {
		return data, nil
	} else if val, ok := data["username"]; ok {
		data["id"] = val
	} else {
		return data, errors.New("Can't retrieve id from the secret, id or username should be specified mandatory")
	}
	return data, nil
}

func NewJenkinsUser(data map[string][]byte, credentialsType string) (JenkinsCredentials, error) {
	out := JenkinsCredentials{}
	crMap, err := generateCredentialsMap(data)
	if err != nil {
		return out, err
	}

	switch credentialsType {
	case SSHUserType:
		params := createSshUserParams(crMap)
		return JenkinsCredentials{Credentials: params}, nil
	case PasswordUserType:
		params := createUserWithPassword(crMap)
		return JenkinsCredentials{Credentials: params}, nil
	case TokenUserType:
		params := createStringCredentials(crMap)
		return JenkinsCredentials{Credentials: params}, nil
	default:
		return out, errors.New("Unknown credentials type!")
	}

}

func (user JenkinsCredentials) ToString() (string, error) {
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

func createUserWithPassword(data map[string]string) JenkinsCredentialsParams {
	username := data["username"]
	password := data["password"]
	description := fmt.Sprintf("%s %s", data["first_name"], data["last_name"])
	return JenkinsCredentialsParams{
		Id:           data["id"],
		Scope:        GlobalScope,
		Username:     &username,
		Password:     &password,
		Description:  &description,
		StaplerClass: "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
	}
}

func createSshUserParams(data map[string]string) JenkinsCredentialsParams {
	username := data["username"]
	password := data["password"]
	description := fmt.Sprintf("%s %s", data["first_name"], data["last_name"])
	return JenkinsCredentialsParams{
		Id:          data["id"],
		Scope:       GlobalScope,
		Username:    &username,
		Password:    &password,
		Description: &description,
		PrivateKeySource: &PrivateKeySource{
			PrivateKey:   data["id_rsa"],
			StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
		},
		StaplerClass: "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey",
	}
}

func createStringCredentials(data map[string]string) JenkinsCredentialsParams {
	secret := data["secret"]
	description := data["username"]
	return JenkinsCredentialsParams{
		Id:           data["id"],
		Scope:        GlobalScope,
		Secret:       &secret,
		Description:  &description,
		StaplerClass: "org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl",
	}
}

func trimNewline(data map[string][]byte) map[string]string {
	out := map[string]string{}
	for k, v := range data {
		out[k] = strings.TrimSuffix(string(v), "\n")
	}
	return out
}

func GetPlatformTypeEnv() string {
	platformType, found := os.LookupEnv(platformType)
	if !found {
		panic("Environment variable PLATFORM_TYPE is not defined")
	}
	return platformType
}

func GetSlavesList(slaves string) []string {
	re := regexp.MustCompile(`\[(.*)\]`)
	if len(re.FindStringSubmatch(slaves)) > 0 {
		return strings.Split(re.FindStringSubmatch(slaves)[1], ", ")
	}

	return nil
}
