package platform

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

var plog = ctrl.Log.WithName("platform_util")

func GetStageInstanceOwner(c client.Client, jj *jenkinsApi.JenkinsJob) (*cdPipeApi.Stage, error) {
	plog.V(2).Info("start getting stage owner cr", "stage", jj.Name)

	if ow := GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		plog.V(2).Info("trying to fetch stage owner from reference", "stage", ow.Name)

		return GetStageInstance(c, ow.Name, jj.Namespace)
	}

	if jj.Spec.StageName != nil {
		plog.V(2).Info("trying to fetch stage owner from spec", "stage", jj.Spec.StageName)

		return GetStageInstance(c, *jj.Spec.StageName, jj.Namespace)
	}

	return nil, fmt.Errorf("failed to find stage owner for jenkins job %v", jj.Name)
}

func GetOwnerReference(ownerKind string, ors []metav1.OwnerReference) *metav1.OwnerReference {
	plog.V(2).Info("finding owner", "kind", ownerKind)

	if len(ors) == 0 {
		return nil
	}

	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o
		}
	}

	return nil
}

func GetStageInstance(c client.Client, name, namespace string) (*cdPipeApi.Stage, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &cdPipeApi.Stage{}

	if err := c.Get(context.TODO(), nsn, i); err != nil {
		return nil, fmt.Errorf("failed to get instance by name %v: %w", name, err)
	}

	return i, nil
}

func GetJenkinsInstanceOwner(c client.Client, name, namespace string, ownerName *string,
	ors []metav1.OwnerReference,
) (*jenkinsApi.Jenkins, error) {
	plog.V(2).Info("start getting jenkins owner", "owner name", name)

	if ow := GetOwnerReference(consts.JenkinsKind, ors); ow != nil {
		plog.V(2).Info("trying to fetch jenkins owner from reference", "jenkins name", ow.Name)

		return GetJenkinsInstance(c, ow.Name, namespace)
	}

	if ownerName != nil {
		plog.V(2).Info("trying to fetch jenkins owner from spec", "jenkins name", ownerName)

		return GetJenkinsInstance(c, *ownerName, namespace)
	}

	plog.V(2).Info("trying to fetch first jenkins instance", "namespace", namespace)

	j, err := GetFirstJenkinsInstance(c, namespace)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func GetJenkinsInstance(c client.Client, name, namespace string) (*jenkinsApi.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &jenkinsApi.Jenkins{}

	if err := c.Get(context.TODO(), nsn, instance); err != nil {
		return nil, fmt.Errorf("failed to get jenkins instance by name %v: %w", name, err)
	}

	return instance, nil
}

func GetFirstJenkinsInstance(c client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	list := &jenkinsApi.JenkinsList{}

	if err := c.List(context.TODO(), list, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, fmt.Errorf("failed to get Jenkins instances in namespace %v: %w", namespace, err)
	}

	if len(list.Items) == 0 {
		return nil, errors.New("at least one Jenkins instance should be accessible")
	}

	j := list.Items[0]

	return GetJenkinsInstance(c, j.Name, j.Namespace)
}

func GetJenkinsFolderInstance(k8sClient client.Client, name, namespace string) (*jenkinsApi.JenkinsFolder, error) {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	jenkinsFolder := &jenkinsApi.JenkinsFolder{}

	if err := k8sClient.Get(context.TODO(), namespacedName, jenkinsFolder); err != nil {
		return nil, fmt.Errorf("failed to get jenkins folder: %w", err)
	}

	return jenkinsFolder, nil
}

// SetControllerReference sets owner as a Controller OwnerReference on owned.
// This is used for garbage collection of the owned object and for
// reconciling the owner object on changes to owned (with a Watch + EnqueueRequestForOwner).
// Since only one OwnerReference can be a controller, it returns an error if
// there is another OwnerReference with Controller flag set.
func SetControllerReference(owner, object metav1.Object, scheme *runtime.Scheme, isController bool) error {
	runtimeObject, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("failed to call SetControllerReference: owner of type %T, should be of type runtime.Object", owner)
	}

	groupVersionKind, err := apiutil.GVKForObject(runtimeObject, scheme)
	if err != nil {
		return fmt.Errorf("failed to get Group Version Kind for object: %w", err)
	}

	// Create a new ref
	ref := *newControllerRef(
		owner,
		schema.GroupVersionKind{
			Group:   groupVersionKind.Group,
			Version: groupVersionKind.Version,
			Kind:    groupVersionKind.Kind,
		},
		isController,
	)

	existingRefs := object.GetOwnerReferences()
	foundIndex := -1

	for i := 0; i < len(existingRefs); i++ {
		if referSameObject(&ref, &existingRefs[i]) {
			foundIndex = i

			break
		}

		if existingRefs[i].Controller != nil && *existingRefs[i].Controller {
			return newAlreadyOwnedError(object, &existingRefs[i])
		}
	}

	if foundIndex == -1 {
		existingRefs = append(existingRefs, ref)
	} else {
		existingRefs[foundIndex] = ref
	}

	// Update owner references
	object.SetOwnerReferences(existingRefs)

	return nil
}

func newControllerRef(owner metav1.Object, gvk schema.GroupVersionKind, isController bool) *metav1.OwnerReference {
	blockOwnerDeletion := true
	or := &metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
	}

	if isController {
		or.Controller = &isController
	}

	return or
}

// Returns true if a and b point to the same object.
func referSameObject(a, b *metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV == bGV && a.Kind == b.Kind && a.Name == b.Name
}

func newAlreadyOwnedError(object metav1.Object, owner *metav1.OwnerReference) *controllerutil.AlreadyOwnedError {
	return &controllerutil.AlreadyOwnedError{
		Object: object,
		Owner:  *owner,
	}
}
