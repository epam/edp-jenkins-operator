package client

import (
	"context"
	jenkinsV1api "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

//var k8sConfig clientcmd.ClientConfig
var SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}

type EdpV1Client struct {
	crClient *rest.RESTClient
}

type JenkinsServiceAccountInterface interface {
	Get(name string, namespace string, options metav1.GetOptions) (result *jenkinsV1api.JenkinsServiceAccount, err error)
	Create(jsa *jenkinsV1api.JenkinsServiceAccount, namespace string) (result *jenkinsV1api.JenkinsServiceAccount, err error)
	Update(jsa *jenkinsV1api.JenkinsServiceAccount) (result *jenkinsV1api.JenkinsServiceAccount, err error)
}

func NewForConfig(config *rest.Config) (*EdpV1Client, error) {
	if err := createCrdClient(config); err != nil {
		return nil, err
	}
	crClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	return &EdpV1Client{crClient: crClient}, nil
}

func (c *EdpV1Client) Get(ctx context.Context, name string, namespace string, options metav1.GetOptions) (result *jenkinsV1api.JenkinsServiceAccount, err error) {
	result = &jenkinsV1api.JenkinsServiceAccount{}
	err = c.crClient.Get().
		Namespace(namespace).
		Resource("jenkinsserviceaccounts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// Create takes the representation of a jenkinsserviceaccount and creates it.  Returns the server's representation of the jenkinsserviceaccount, and an error, if there is any.
func (c *EdpV1Client) Create(ctx context.Context, jsa *jenkinsV1api.JenkinsServiceAccount, namespace string) (result *jenkinsV1api.JenkinsServiceAccount, err error) {
	result = &jenkinsV1api.JenkinsServiceAccount{}
	err = c.crClient.Post().
		Namespace(namespace).
		Resource("jenkinsserviceaccounts").
		Body(jsa).
		Do(ctx).
		Into(result)
	return
}

func (c *EdpV1Client) Update(ctx context.Context, jsa *jenkinsV1api.JenkinsServiceAccount) (result *jenkinsV1api.JenkinsServiceAccount, err error) {
	result = &jenkinsV1api.JenkinsServiceAccount{}
	err = c.crClient.Put().
		Namespace(jsa.Namespace).
		Resource("jenkinsserviceaccounts").
		Name(jsa.Name).
		Body(jsa).
		Do(ctx).
		Into(result)
	return
}

func createCrdClient(cfg *rest.Config) error {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	config := cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)

	return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&jenkinsV1api.JenkinsServiceAccount{},
		&jenkinsV1api.JenkinsServiceAccountList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
