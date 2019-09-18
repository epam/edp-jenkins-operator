package client

import (
	jenkinsV1api "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
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

type JenkinsInterface interface {
	Get(name string, namespace string, options metav1.GetOptions) (result *jenkinsV1api.Jenkins, err error)
	Create(jsa *jenkinsV1api.Jenkins, namespace string) (result *jenkinsV1api.Jenkins, err error)
	Update(jsa *jenkinsV1api.Jenkins) (result *jenkinsV1api.Jenkins, err error)
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

func (c *EdpV1Client) Get(name string, namespace string, options metav1.GetOptions) (result *jenkinsV1api.Jenkins, err error) {
	result = &jenkinsV1api.Jenkins{}
	err = c.crClient.Get().
		Namespace(namespace).
		Resource("jenkins").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Create takes the representation of a Jenkins and creates it.  Returns the server's representation of the Jenkins, and an error, if there is any.
func (c *EdpV1Client) Create(jsa *jenkinsV1api.Jenkins, namespace string) (result *jenkinsV1api.Jenkins, err error) {
	result = &jenkinsV1api.Jenkins{}
	err = c.crClient.Post().
		Namespace(namespace).
		Resource("jenkins").
		Body(jsa).
		Do().
		Into(result)
	return
}

func (c *EdpV1Client) Update(jsa *jenkinsV1api.Jenkins) (result *jenkinsV1api.Jenkins, err error) {
	result = &jenkinsV1api.Jenkins{}
	err = c.crClient.Put().
		Namespace(jsa.Namespace).
		Resource("jenkins").
		Name(jsa.Name).
		Body(jsa).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Jenkins that match those selectors.
func (c *EdpV1Client) List(opts metav1.ListOptions, namespace string) (result *jenkinsV1api.JenkinsList, err error) {
	result = &jenkinsV1api.JenkinsList{}
	err = c.crClient.Get().
		Namespace(namespace).
		Resource("jenkins").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
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
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&jenkinsV1api.Jenkins{},
		&jenkinsV1api.JenkinsList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
