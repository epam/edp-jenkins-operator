package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsSpec defines the desired state of Jenkins.
type JenkinsSpec struct {
	// RestAPIUrl jenkins full rest api url
	RestAPIUrl string `json:"restAPIUrl,omitempty"`
	// ExternalURL jenkins full external url for keycloak or other integrations
	ExternalURL string `json:"externalURL,omitempty"`
	// +optional
	BasePath string `json:"basePath,omitempty"`
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

// JenkinsStatus defines the observed state of Jenkins.
type JenkinsStatus struct {
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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Jenkins is the Schema for the jenkins API.
type Jenkins struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec JenkinsSpec `json:"spec,omitempty"`
	// +optional
	Status JenkinsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsList contains a list of Jenkins.
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}
