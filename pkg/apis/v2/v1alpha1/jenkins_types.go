package v1alpha1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JenkinsSpec defines the desired state of Jenkins
// +k8s:openapi-gen=true

type JenkinsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Image     string `json:"image"`
	Version   string `json:"version"`
	InitImage string `json:"initImage"`
	// +optional
	BasePath string `json:"basePath,omitempty"`
	// +nullable
	// +optional
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +nullable
	// +optional
	Volumes []JenkinsVolumes `json:"volumes,omitempty"`
	// +nullable
	// +optional
	SharedLibraries []JenkinsSharedLibraries `json:"sharedLibraries,omitempty"`
	KeycloakSpec    KeycloakSpec             `json:"keycloakSpec"`
	// +optional
	EdpSpec EdpSpec `json:"edpSpec,omitempty"`
}

type EdpSpec struct {
	DnsWildcard string `json:"dnsWildcard"`
}

type JenkinsVolumes struct {
	Name         string `json:"name"`
	StorageClass string `json:"storageClass"`
	Capacity     string `json:"capacity"`
}

type JenkinsSharedLibraries struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Tag  string `json:"tag"`
	// +nullable
	// +optional
	CredentialID *string `json:"secret,omitempty"`
	// +nullable
	// +optional
	Type *string `json:"type,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
// +k8s:openapi-gen=true
type JenkinsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	// +optional
	Status string `json:"status,omitempty"`
	// +optional
	AdminSecretName string `json:"adminSecretName,omitempty"`
	// +nullable
	// +optional
	Slaves []Slave `json:"slaves,omitempty"`
	// +nullable
	// +optional
	JobProvisions []JobProvision `json:"jobProvisions,omitempty"`
}

type Slave struct {
	// +optional
	Name string `json:"name,omitempty"`
}

type JobProvision struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type KeycloakSpec struct {
	Enabled bool `json:"enabled"`
	// +optional
	Realm string `json:"realm,omitempty"`
	// +optional
	IsPrivate bool `json:"isPrivate,omitempty"`
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:deprecatedversion

// Jenkins is the Schema for the jenkins API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Jenkins struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec JenkinsSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JenkinsList contains a list of Jenkins.
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}
