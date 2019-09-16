package helper

import (
	"context"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const JenkinsDefaultScriptConfigMapKey = "context"


//TODO(Serhii_Shydlovskyi): Remove this, after refactoring in other operators.
type K8sClient struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func CreateJenkinsScript(runtimeClient K8sClient, name string, configMapName string, namespace string, setOwnerReference bool, ownerInstance *v1alpha1.Jenkins) (*v1alpha1.JenkinsScript, error) {
	jenkinsScriptObject := &v1alpha1.JenkinsScript{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.JenkinsScriptSpec{
			SourceCmName: configMapName,
		},
	}

	if setOwnerReference {
		if err := controllerutil.SetControllerReference(ownerInstance, jenkinsScriptObject, runtimeClient.Scheme); err != nil {
			return nil, errors.Wrapf(err, "Couldn't set reference for JenkinsScript %v object", jenkinsScriptObject.Name)
		}
	}

	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := runtimeClient.Client.Get(context.TODO(), nsn, jenkinsScriptObject)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			err := runtimeClient.Client.Create(context.TODO(), jenkinsScriptObject)
			if err != nil {
				return nil, errors.Wrapf(err, "Couldn't create Jenkins Script object %v", name)
			}
		}
	}

	return jenkinsScriptObject, nil
}
