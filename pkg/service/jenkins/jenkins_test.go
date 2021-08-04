package jenkins

import (
	"os"
	"strings"
	"testing"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/pkg/errors"
)

func TestJenkinsServiceImpl_createTemplateScript(t *testing.T) {
	ji := v1alpha1.Jenkins{}
	platformMock := platform.Mock{}
	jenkinsScriptData := platformHelper.JenkinsScriptData{}

	fp, err := os.Create("/tmp/temp.tpl")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fp.WriteString("lol"); err != nil {
		t.Fatal(err)
	}

	if err := fp.Close(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove("/tmp/temp.tpl"); err != nil {
			t.Fatal(err)
		}
	}()

	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp").Return(false, nil)
	platformMock.On("CreateJenkinsScript", "", "-temp", false).Return(&v1alpha1.JenkinsScript{}, nil)

	if err := createTemplateScript("/tmp", "temp.tpl", &platformMock, jenkinsScriptData, ji); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestJenkinsServiceImpl_createTemplateScript_Failure(t *testing.T) {
	ji := v1alpha1.Jenkins{
		Spec: v1alpha1.JenkinsSpec{
			Version: "0",
		},
	}
	platformMock := platform.Mock{}
	jenkinsScriptData := platformHelper.JenkinsScriptData{}

	err := createTemplateScript("/tmp", "temp123.tpl", &platformMock, jenkinsScriptData,
		ji)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "Template file not found in pathToTemplate specificed") {
		t.Log(errors.Cause(err).Error())
		t.Fatal("wrong error returned")
	}

	fp, err := os.Create("/tmp/temp.tpl")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fp.WriteString("lol"); err != nil {
		t.Fatal(err)
	}

	if err := fp.Close(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove("/tmp/temp.tpl"); err != nil {
			t.Fatal(err)
		}
	}()

	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp").
		Return(false, errors.New("CreateConfigMap fatal"))

	err = createTemplateScript("/tmp", "temp.tpl", &platformMock, jenkinsScriptData, ji)

	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "CreateConfigMap fatal") {
		t.Log(errors.Cause(err).Error())
		t.Fatal("wrong error returned")
	}

	ji.Spec.Version = "2"
	platformMock.On("CreateConfigMapWithUpdate", ji, "-temp").
		Return(false, nil)
	platformMock.On("CreateJenkinsScript", "", "-temp", false).
		Return(nil, errors.New("CreateJenkinsScript fatal"))

	err = createTemplateScript("/tmp", "temp.tpl", &platformMock, jenkinsScriptData, ji)

	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "CreateJenkinsScript fatal") {
		t.Log(errors.Cause(err).Error())
		t.Fatal("wrong error returned")
	}
}
