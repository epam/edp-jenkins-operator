package helper

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestNewJenkinsUser(t *testing.T) {
	data := map[string][]byte{}
	secretName := "name"
	credentialsType := []string{SSHUserType, PasswordUserType, TokenUserType}
	for i := range credentialsType {
		_, err := NewJenkinsUser(data, credentialsType[i], secretName)
		assert.NoError(t, err)
	}
}

func TestNewJenkinsUserErr(t *testing.T) {
	data := map[string][]byte{}
	secretName := "name"
	credentialsType := ""
	_, err := NewJenkinsUser(data, credentialsType, secretName)
	assert.Equal(t, "Unknown credentials type!", err.Error())
}

func TestJenkinsCredentials_ToString(t *testing.T) {
	credentials := JenkinsCredentials{
		Credentials: JenkinsCredentialsParams{Id: "1"},
	}
	str, err := credentials.ToString()
	assert.NoError(t, err)
	assert.Equal(t, "{\"credentials\":{\"id\":\"1\",\"scope\":\"\",\"stapler-class\":\"\"}}", str)
}

func TestGetPlatformTypeEnv(t *testing.T) {
	str := "test"
	err := os.Setenv(PlatformType, str)
	assert.NoError(t, err)
	env, err := GetPlatformTypeEnv()
	assert.NoError(t, err)
	assert.Equal(t, str, env)
	err = os.Unsetenv(PlatformType)
	assert.NoError(t, err)
}

func TestGetPlatformTypeEnvErr(t *testing.T) {
	env, err := GetPlatformTypeEnv()
	assert.Error(t, err)
	assert.Equal(t, "", env)
}

func TestNewTrue(t *testing.T) {
	assert.True(t, *NewTrue())
}

func TestTryToDelete_AddFinalizers(t *testing.T) {
	ja := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ja1",
			Namespace: "ns1",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	if _, err := TryToDelete(&ja, "fint", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(ja.GetFinalizers()) == 0 {
		t.Fatal("no finalizers added")
	}
}

func TestTryToDelete_RemoveFinalizers(t *testing.T) {
	ja := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ja1",
			Namespace:         "ns1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"fint"},
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	if _, err := TryToDelete(&ja, "fint", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(ja.GetFinalizers()) > 0 {
		t.Fatal("finalizers is not removed")
	}
}

func TestTryToDelete_DeleteFuncFailure(t *testing.T) {
	ja := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ja1",
			Namespace:         "ns1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	_, err := TryToDelete(&ja, "fint", func() error {
		return errors.New("del func fatal")
	})
	if err == nil {
		t.Fatal("no error func returned")
	}

	if !strings.Contains(err.Error(), "del func fatal") {
		t.Log(err)
		t.Fatal("wrong func returned")
	}
}

func TestJenkinsIsNotFoundErr(t *testing.T) {
	err := errors.New("404")
	assert.True(t, JenkinsIsNotFoundErr(err))
}
