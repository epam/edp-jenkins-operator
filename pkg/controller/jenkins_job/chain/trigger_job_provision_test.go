package chain

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	common "github.com/epam/edp-common/pkg/mock"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

func TestTriggerJobProvision_ServeRequest_triggerJobProvisionErr(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	platform := pmock.PlatformService{}
	jenkinsJob := &v1alpha1.JenkinsJob{}
	logger := common.Logger{}

	trigger := TriggerJobProvision{
		next:   nil,
		client: client,
		ps:     &platform,
		log:    &logger,
	}
	err := trigger.ServeRequest(jenkinsJob)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while updating"))
}

func TestTriggerJobProvision_ServeRequest_triggerJobProvisionErr2(t *testing.T) {
	jenkinsJob := &v1alpha1.JenkinsJob{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.JenkinsJob{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsJob).Build()
	platform := pmock.PlatformService{}
	logger := common.Logger{}

	trigger := TriggerJobProvision{
		next:   nil,
		client: client,
		ps:     &platform,
		log:    &logger,
	}
	err := trigger.ServeRequest(jenkinsJob)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client"))
}

func TestTriggerJobProvision_ServeRequest(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	ownerReference := v1.OwnerReference{Kind: "Jenkins", Name: name}

	data := map[string]string{"str1": "str2"}
	raw, err := json.Marshal(data)
	assert.NoError(t, err)

	jenkinsJob := &v1alpha1.JenkinsJob{
		ObjectMeta: v1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []v1.OwnerReference{ownerReference},
		},
		Spec: v1alpha1.JenkinsJobSpec{
			Job: v1alpha1.Job{
				Config: string(raw),
			},
		},
	}

	jenkins := &v1alpha1.Jenkins{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.JenkinsJob{}, &v1alpha1.Jenkins{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsJob, jenkins).Build()
	platform := pmock.PlatformService{}
	logger := common.Logger{}

	// look at taskResponse struct from goJenkins queue.go
	taskResponseRaw := []byte("{\"executable\":{\"number\":1,\"url\":\"\"}}")

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)
	httpmock.RegisterResponder(http.MethodGet, "https://api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://queue/item/0/api/json", httpmock.NewBytesResponder(http.StatusOK, taskResponseRaw))

	trigger := TriggerJobProvision{
		next:   nil,
		client: client,
		ps:     &platform,
		log:    &logger,
	}
	err = trigger.ServeRequest(jenkinsJob)
	assert.NoError(t, err)
}
