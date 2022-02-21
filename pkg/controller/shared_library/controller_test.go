package sharedLibrary

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
)

type SharedLibraryTestSuite struct {
	suite.Suite
	rootGerrit              *v1alpha1.Jenkins
	library, anotherLibrary *v1alpha1.JenkinsSharedLibrary
	logger                  *helper.LoggerMock
	scheme                  *runtime.Scheme
	fakeClient              client.Client
	platformService         *mock.PlatformService
}

func TestSharedLibraryTestSuite(t *testing.T) {
	suite.Run(t, new(SharedLibraryTestSuite))
}

func (s *SharedLibraryTestSuite) SetupTest() {
	s.rootGerrit = &v1alpha1.Jenkins{TypeMeta: metav1.TypeMeta{Kind: "Jenkins", APIVersion: "v2.edp.epam.com/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns1", Name: "ger"},
		Status:     v1alpha1.JenkinsStatus{Status: "ready"}}
	s.library = &v1alpha1.JenkinsSharedLibrary{
		ObjectMeta: metav1.ObjectMeta{Name: "lib1", Namespace: s.rootGerrit.Namespace},
		Spec:       v1alpha1.JenkinsSharedLibrarySpec{OwnerName: &s.rootGerrit.Name},
	}
	s.anotherLibrary = s.library.DeepCopy()
	s.anotherLibrary.Name = "lib2"

	s.logger = &helper.LoggerMock{}
	s.scheme = runtime.NewScheme()
	assert.NoError(s.T(), v1alpha1.AddToScheme(s.scheme))
	s.fakeClient = fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.library, s.rootGerrit,
		s.anotherLibrary).Build()
	s.platformService = &mock.PlatformService{}

	fp, err := os.Create(jenkins.SharedLibrariesTemplateName)
	assert.NoError(s.T(), err)
	err = fp.Close()
	assert.NoError(s.T(), err)
}

func (s *SharedLibraryTestSuite) TearDownTest() {
	s.platformService.AssertExpectations(s.T())

	err := os.RemoveAll(jenkins.SharedLibrariesTemplateName)
	assert.NoError(s.T(), err)
}

func (s *SharedLibraryTestSuite) TestReconcile() {
	r := Reconcile{
		client:          s.fakeClient,
		log:             s.logger,
		platformService: s.platformService,
	}

	s.platformService.On("CreateConfigMapWithUpdate", s.rootGerrit, scriptConfigMapName,
		map[string]string{"context": ""}).Return(false, nil)
	s.platformService.On("CreateJenkinsScript", s.rootGerrit.Namespace, scriptConfigMapName, false).
		Return(&v1alpha1.JenkinsScript{}, nil)

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: s.library.Namespace, Name: s.library.Name},
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, time.Duration(0))

	assert.NoError(s.T(), s.logger.LastError())
}

func (s *SharedLibraryTestSuite) TestJenkinsNotReady() {
	rootGerrit := s.rootGerrit.DeepCopy()
	rootGerrit.Status.Status = ""

	r := Reconcile{
		client:          fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(rootGerrit, s.library).Build(),
		log:             s.logger,
		platformService: s.platformService,
	}

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: s.library.Namespace, Name: s.library.Name},
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, helper.DefaultRequeueTime*time.Second)
	assert.EqualError(s.T(), s.logger.LastError(), "root jenkins is not ready")
}

func (s *SharedLibraryTestSuite) TestLibraryNotFound() {
	r := Reconcile{
		client:          fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.rootGerrit).Build(),
		log:             s.logger,
		platformService: s.platformService,
	}

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: s.library.Namespace, Name: s.library.Name},
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, time.Duration(0))
	assert.NoError(s.T(), s.logger.LastError())
	assert.Equal(s.T(), s.logger.LastInfo(), "instance not found")
}

func (s *SharedLibraryTestSuite) TestRootJenkinsNotFound() {
	r := Reconcile{
		client:          fake.NewClientBuilder().WithScheme(s.scheme).WithRuntimeObjects(s.library).Build(),
		log:             s.logger,
		platformService: s.platformService,
	}

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: s.library.Namespace, Name: s.library.Name},
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, helper.DefaultRequeueTime*time.Second)

	assert.EqualError(s.T(), s.logger.LastError(),
		"an error has been occurred while getting owner jenkins for jenkins folder lib1: failed to get jenkins instance by name ger: jenkinses.v2.edp.epam.com \"ger\" not found")
}

func (s *SharedLibraryTestSuite) TestReconcileFailure() {
	r := Reconcile{
		client:          s.fakeClient,
		log:             s.logger,
		platformService: s.platformService,
	}

	s.platformService.On("CreateConfigMapWithUpdate", s.rootGerrit, scriptConfigMapName,
		map[string]string{"context": ""}).Return(false, nil)
	s.platformService.On("CreateJenkinsScript", s.rootGerrit.Namespace, scriptConfigMapName, false).
		Return(nil, errors.New("fatal"))

	res, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: s.library.Namespace, Name: s.library.Name},
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.RequeueAfter, helper.DefaultRequeueTime*time.Second)

	assert.EqualError(s.T(), s.logger.LastError(),
		"unable to create libraries script: unable to create jenkins script: fatal")
}

func TestSpecIsUpdated(t *testing.T) {
	assert.False(t, specUpdated(event.UpdateEvent{
		ObjectOld: &v1alpha1.JenkinsSharedLibrary{},
		ObjectNew: &v1alpha1.JenkinsSharedLibrary{},
	}))
}
