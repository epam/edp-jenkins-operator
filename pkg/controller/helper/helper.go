package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
)

const (
	DefaultRequeueTime = 30
	GlobalScope        = "GLOBAL"
	SSHUserType        = "ssh"
	PasswordUserType   = "password"
	TokenUserType      = "token"
	PlatformType       = "PLATFORM_TYPE"
	StatusSuccess      = "success"

	idKey = "id"
)

func NewTrue() *bool {
	value := true
	return &value
}

func generateCredentialsMap(rawData map[string][]byte, secretName string) (map[string]string, error) {
	data := trimNewline(rawData)
	if _, ok := data[idKey]; ok {
		return data, nil
	}

	data[idKey] = secretName

	return data, nil
}

type JenkinsCredentials struct {
	Credentials JenkinsCredentialsParams `json:"credentials"`
}

func NewJenkinsUser(data map[string][]byte, credentialsType, secretName string) (JenkinsCredentials, error) {
	out := JenkinsCredentials{}

	crMap, err := generateCredentialsMap(data, secretName)
	if err != nil {
		return out, err
	}

	switch credentialsType {
	case SSHUserType:
		params := createSshUserParams(crMap, secretName)

		return JenkinsCredentials{Credentials: params}, nil
	case PasswordUserType:
		params := createUserWithPassword(crMap, secretName)

		return JenkinsCredentials{Credentials: params}, nil
	case TokenUserType:
		params := createStringCredentials(crMap, secretName)

		return JenkinsCredentials{Credentials: params}, nil
	default:
		return out, errors.New("unknown credentials type")
	}
}

func (user *JenkinsCredentials) ToString() (string, error) {
	bytes, err := json.Marshal(*user)
	if err != nil {
		return "", fmt.Errorf("failed to convert jenkins credentials to json: %w", err)
	}

	return string(bytes), nil
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

func createUserWithPassword(data map[string]string, secretName string) JenkinsCredentialsParams {
	username := data["username"]
	password := data["password"]

	if data[idKey] == "" {
		data[idKey] = secretName
	}

	description := data[idKey]

	return JenkinsCredentialsParams{
		Id:           data[idKey],
		Scope:        GlobalScope,
		Username:     &username,
		Password:     &password,
		Description:  &description,
		StaplerClass: "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
	}
}

func createSshUserParams(data map[string]string, secretName string) JenkinsCredentialsParams {
	username := data["username"]
	password := data["password"]

	if data[idKey] == "" {
		data[idKey] = secretName
	}

	description := data[idKey]

	return JenkinsCredentialsParams{
		Id:          data[idKey],
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

func createStringCredentials(data map[string]string, secretName string) JenkinsCredentialsParams {
	secret := data["secret"]

	if data[idKey] == "" {
		data[idKey] = secretName
	}

	description := data[idKey]

	return JenkinsCredentialsParams{
		Id:           data[idKey],
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

func GetPlatformTypeEnv() (string, error) {
	platformType, found := os.LookupEnv(PlatformType)
	if !found {
		return "", errors.New("environment variable PLATFORM_TYPE not found")
	}

	return platformType, nil
}

func GetSlavesList(slaves string) []string {
	re := regexp.MustCompile(`\[(.*)\]`)

	if len(re.FindStringSubmatch(slaves)) > 0 {
		return strings.Split(re.FindStringSubmatch(slaves)[1], ", ")
	}

	return nil
}

func TryToDelete(instance client.Object, finalizerName string, deleteFunc func() error) (needUpdate bool, err error) {
	if instance.GetDeletionTimestamp().IsZero() {
		finalizers := instance.GetFinalizers()

		if !finalizer.ContainsString(finalizers, finalizerName) {
			finalizers = append(finalizers, finalizerName)
			needUpdate = true

			instance.SetFinalizers(finalizers)
		}

		return
	}

	if err := deleteFunc(); err != nil {
		return false, fmt.Errorf("failed to perform delete function: %w", err)
	}

	finalizers := instance.GetFinalizers()
	finalizers = finalizer.RemoveString(finalizers, finalizerName)

	instance.SetFinalizers(finalizers)

	return true, nil
}

func JenkinsIsNotFoundErr(err error) bool {
	return err.Error() == "404"
}
