// NOTE: Boilerplate only.  Ignore this file.

// Package v1 contains API Schema definitions for the v2.edp.epam.com API group
// +k8s:deepcopy-gen=package,register
// +groupName=v2.edp.epam.com
package v1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

func AddToScheme(sch *runtime.Scheme) error {
	SchemeBuilder.Register(&JenkinsJob{}, &JenkinsJobList{},
		&CDStageJenkinsDeployment{}, &CDStageJenkinsDeploymentList{},
		&Jenkins{}, &JenkinsList{},
		&JenkinsAgent{}, &JenkinsAgentList{},
		&JenkinsAuthorizationRole{}, &JenkinsAuthorizationRoleList{},
		&JenkinsAuthorizationRoleMapping{}, &JenkinsAuthorizationRoleMappingList{},
		&JenkinsFolder{}, &JenkinsFolderList{},
		&JenkinsJobBuildRun{}, &JenkinsJobBuildRunList{},
		&JenkinsScript{}, &JenkinsScriptList{},
		&JenkinsServiceAccount{}, &JenkinsServiceAccountList{},
		&JenkinsSharedLibrary{}, &JenkinsSharedLibraryList{})

	if err := SchemeBuilder.AddToScheme(sch); err != nil {
		return errors.Wrap(err, "error during scheme building")
	}

	return nil
}
