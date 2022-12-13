package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}

type EdpV1Client struct {
	crClient *rest.RESTClient
}

type JenkinsServiceAccountInterface interface {
	Get(name string, namespace string, options metav1.GetOptions) (result *jenkinsApi.JenkinsServiceAccount, err error)
	Create(jsa *jenkinsApi.JenkinsServiceAccount, namespace string) (result *jenkinsApi.JenkinsServiceAccount, err error)
	Update(jsa *jenkinsApi.JenkinsServiceAccount) (result *jenkinsApi.JenkinsServiceAccount, err error)
}

func NewForConfig(config *rest.Config) (*EdpV1Client, error) {
	if err := createCrdClient(config); err != nil {
		return nil, err
	}

	crClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &EdpV1Client{crClient: crClient}, nil
}

func (c *EdpV1Client) Get(ctx context.Context, name, namespace string, options metav1.GetOptions) (*jenkinsApi.JenkinsServiceAccount, error) {
	result := &jenkinsApi.JenkinsServiceAccount{}

	err := c.crClient.Get().
		Namespace(namespace).
		Resource("jenkinsserviceaccounts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	if err != nil {
		return result, fmt.Errorf("failed to get JenkinsServiceaccount: %w", err)
	}

	return result, nil
}

// Create takes the representation of a jenkinsserviceaccount and creates it.  Returns the server's representation of the jenkinsserviceaccount, and an error, if there is any.
func (c *EdpV1Client) Create(ctx context.Context, jsa *jenkinsApi.JenkinsServiceAccount, namespace string) (*jenkinsApi.JenkinsServiceAccount, error) {
	result := &jenkinsApi.JenkinsServiceAccount{}

	err := c.crClient.Post().
		Namespace(namespace).
		Resource("jenkinsserviceaccounts").
		Body(jsa).
		Do(ctx).
		Into(result)
	if err != nil {
		return nil, fmt.Errorf("failed to Post JenkinsServiceAccount: %w", err)
	}

	return result, nil
}

func (c *EdpV1Client) Update(ctx context.Context, jsa *jenkinsApi.JenkinsServiceAccount) (*jenkinsApi.JenkinsServiceAccount, error) {
	result := &jenkinsApi.JenkinsServiceAccount{}

	err := c.crClient.Put().
		Namespace(jsa.Namespace).
		Resource("jenkinsserviceaccounts").
		Name(jsa.Name).
		Body(jsa).
		Do(ctx).
		Into(result)
	if err != nil {
		return result, fmt.Errorf("failed to Put JenkinsServiceAccount: %w", err)
	}

	return result, nil
}

func createCrdClient(cfg *rest.Config) error {
	runtimeScheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)

	if err := SchemeBuilder.AddToScheme(runtimeScheme); err != nil {
		return fmt.Errorf("failed to add runtime scheme: %w", err)
	}

	config := cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtimeScheme)

	return nil
}

func addKnownTypes(runtimeScheme *runtime.Scheme) error {
	runtimeScheme.AddKnownTypes(SchemeGroupVersion,
		&jenkinsApi.JenkinsServiceAccount{},
		&jenkinsApi.JenkinsServiceAccountList{},
	)

	metav1.AddToGroupVersion(runtimeScheme, SchemeGroupVersion)

	return nil
}
