package apis

import (
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
