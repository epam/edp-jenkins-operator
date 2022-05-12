package v1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JenkinsSpec defines the desired state of Jenkins
type JenkinsSpec struct {
	Image     string `json:"image"`
	Version   string `json:"version"`
	InitImage string `json:"initImage"`
	BasePath  string `json:"basePath,omitempty"`
	// +nullable
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +nullable
	Volumes []JenkinsVolumes `json:"volumes,omitempty"`
	// +nullable
	SharedLibraries []JenkinsSharedLibraries `json:"sharedLibraries,omitempty"`
	KeycloakSpec    KeycloakSpec             `json:"keycloakSpec"`
	EdpSpec         EdpSpec                  `json:"edpSpec,omitempty"`
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
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	Tag          string  `json:"tag"`
	CredentialID *string `json:"secret,omitempty"`
	Type         *string `json:"type,omitempty"`
}

// JenkinsStatus defines the observed state of Jenkins
type JenkinsStatus struct {
	Available       bool        `json:"available,omitempty"`
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	Status          string      `json:"status,omitempty"`
	AdminSecretName string      `json:"adminSecretName,omitempty"`
	// +nullable
	Slaves []Slave `json:"slaves,omitempty"`
	// +nullable
	JobProvisions []JobProvision `json:"jobProvisions,omitempty"`
}

type Slave struct {
	Name string `json:"name,omitempty"`
}

type JobProvision struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type KeycloakSpec struct {
	Enabled    bool   `json:"enabled"`
	Realm      string `json:"realm,omitempty"`
	IsPrivate  bool   `json:"isPrivate,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// Jenkins is the Schema for the jenkins API
type Jenkins struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JenkinsSpec   `json:"spec,omitempty"`
	Status JenkinsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JenkinsList contains a list of Jenkins
type JenkinsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Jenkins `json:"items"`
}
