package chain

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	common "github.com/epam/edp-common/pkg/mock"
	jjmock "github.com/epam/edp-jenkins-operator/v2/mock/jenkins_job"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

const (
	name      = "name"
	namespace = "namespace"
	URLScheme = "https"
)

func ObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func TestPutJenkinsPipeline_ServeRequest_setStatusErr(t *testing.T) {
	jenkinsJob := &jenkinsApi.JenkinsJob{ObjectMeta: ObjectMeta()}

	client := fake.NewClientBuilder().Build()
	jenkinsJobHandler := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jenkinsJobHandler,
		client: client,
		ps:     &platform,
		log:    &lg,
	}

	err := pipeline.ServeRequest(jenkinsJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set status")
}

func TestPutJenkinsPipeline_ServeRequest_tryToCreateJobErr(t *testing.T) {
	jenkinsJob := &jenkinsApi.JenkinsJob{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsJob).Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	err := pipeline.ServeRequest(jenkinsJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get owner jenkins for jenkins job name")
}

func TestPutJenkinsPipeline_ServeRequest_setStatusErr2(t *testing.T) {
	jenkinsJob := &jenkinsApi.JenkinsJob{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsJob).Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	err := pipeline.ServeRequest(jenkinsJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get owner jenkins for jenkins job name")
}

func TestPutJenkinsPipeline_ServeRequest(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()

	orJenkins := v1.OwnerReference{Kind: "Jenkins", Name: name}
	orStage := v1.OwnerReference{Kind: "Stage", Name: name}
	jenkinsJob := &jenkinsApi.JenkinsJob{ObjectMeta: ObjectMeta()}
	jenkinsJob.ObjectMeta.OwnerReferences = []v1.OwnerReference{orJenkins, orStage}
	jenkinsJob.Spec.Job.Name = name
	jenkins := &jenkinsApi.Jenkins{ObjectMeta: ObjectMeta()}
	stage := &cdPipeApi.Stage{
		TypeMeta: v1.TypeMeta{
			Kind:       "Stage",
			APIVersion: "meta.k8s.io/v1",
		},
		ObjectMeta: ObjectMeta(),
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(
		v1.SchemeGroupVersion,
		&jenkinsApi.JenkinsJob{},
		&jenkinsApi.JenkinsList{},
		&jenkinsApi.Jenkins{},
		&cdPipeApi.Stage{},
	)

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(jenkinsJob, jenkins, stage).Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	platform.On("GetExternalEndpoint", namespace, name).Return("", URLScheme, "", nil)
	platform.On("GetSecretData", namespace, "").Return(secretData, nil)

	innerJob := gojenkins.InnerJob{Name: name}
	Raw := gojenkins.ExecutorResponse{Jobs: []gojenkins.InnerJob{innerJob}}

	marshal, err := json.Marshal(Raw)
	assert.NoError(t, err)

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://api/json",
		httpmock.NewBytesResponder(http.StatusOK, marshal),
	)
	platform.On("CreateStageJSON", stage).Return(name, nil)
	httpmock.RegisterResponder(
		http.MethodGet,
		"https://crumbIssuer/api/json/api/json",
		httpmock.NewStringResponder(http.StatusOK, ""),
	)
	httpmock.RegisterResponder(
		http.MethodPost,
		"https://createItem",
		httpmock.NewStringResponder(http.StatusOK, ""),
	)
	jh.On("ServeRequest", jenkinsJob).Return(nil)

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	err = pipeline.ServeRequest(jenkinsJob)
	assert.NoError(t, err)
}

func TestPutJenkinsPipeline_setPipeSrcParams_getLibraryParamsErr(t *testing.T) {
	stage := &cdPipeApi.Stage{
		ObjectMeta: ObjectMeta(),
		Spec: cdPipeApi.StageSpec{
			Source: cdPipeApi.Source{
				Library: cdPipeApi.Library{
					Name: name,
				},
			},
		},
	}
	cl := fake.NewClientBuilder().Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	pipeline.setPipeSrcParams(stage, nil)
	assert.Error(t, lg.LastError())
	assert.Contains(t, lg.LastError().Error(), "Codebase")
}

func TestPutJenkinsPipeline_setPipeSrcParams_getGitServerParamsErr(t *testing.T) {
	stage := &cdPipeApi.Stage{
		ObjectMeta: ObjectMeta(),
		Spec: cdPipeApi.StageSpec{
			Source: cdPipeApi.Source{
				Library: cdPipeApi.Library{
					Name: name,
				},
			},
		},
	}

	cb := &codebaseApi.Codebase{
		ObjectMeta: ObjectMeta(),
		Spec: codebaseApi.CodebaseSpec{
			GitServer: name,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &codebaseApi.Codebase{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cb).Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	pipeline.setPipeSrcParams(stage, nil)
	assert.Error(t, lg.LastError())
	assert.Contains(t, lg.LastError().Error(), "GitServer")
}

func TestPutJenkinsPipeline_setPipeSrcParams(t *testing.T) {
	stage := &cdPipeApi.Stage{
		ObjectMeta: ObjectMeta(),
		Spec: cdPipeApi.StageSpec{
			Source: cdPipeApi.Source{
				Library: cdPipeApi.Library{
					Name: name,
				},
			},
		},
	}

	cb := &codebaseApi.Codebase{
		ObjectMeta: ObjectMeta(),
		Spec: codebaseApi.CodebaseSpec{
			GitServer: name,
		},
	}

	ps := map[string]interface{}{}

	gs := &codebaseApi.GitServer{ObjectMeta: ObjectMeta()}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &codebaseApi.Codebase{}, &codebaseApi.GitServer{})

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cb, gs).Build()
	jh := jjmock.JenkinsJobHandler{}
	platform := pmock.PlatformService{}
	lg := common.Logger{}

	pipeline := PutJenkinsPipeline{
		next:   &jh,
		client: cl,
		ps:     &platform,
		log:    &lg,
	}

	pipeline.setPipeSrcParams(stage, ps)
	assert.NoError(t, lg.LastError())
}

func Test_getResult(t *testing.T) {
	t.Parallel()

	type args struct {
		status string
	}

	tests := []struct {
		name string
		args args
		want jenkinsApi.Result
	}{
		{
			name: "should return \"success\"",
			args: args{
				status: consts.StatusFinished,
			},
			want: "success",
		},
		{
			name: "should return \"error\"",
			args: args{
				status: consts.StatusFailed,
			},
			want: "error",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getResult(tt.args.status)

			assert.Equal(t, tt.want, got)
		})
	}
}
