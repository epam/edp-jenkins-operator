package platform

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var plog = logf.Log.WithName("platform_util")

func GetStageInstanceOwner(c client.Client, jj v1alpha1.JenkinsJob) (*edpv1alpha1.Stage, error) {
	plog.V(2).Info("start getting stage owner cr", "stage", jj.Name)
	if ow := GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		plog.V(2).Info("trying to fetch stage owner from reference", "stage", ow.Name)
		return GetStageInstance(c, ow.Name, jj.Namespace)
	}
	if jj.Spec.StageName != nil {
		plog.V(2).Info("trying to fetch stage owner from spec", "stage", jj.Spec.StageName)
		return GetStageInstance(c, *jj.Spec.StageName, jj.Namespace)
	}
	return nil, fmt.Errorf("couldn't find stage owner for jenkins job %v", jj.Name)
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

func GetStageInstance(c client.Client, name, namespace string) (*edpv1alpha1.Stage, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &edpv1alpha1.Stage{}
	if err := c.Get(context.TODO(), nsn, i); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by name %v", name)
	}
	return i, nil
}

func GetJenkinsInstanceOwner(c client.Client, name, namespace string, ownerName *string,
	ors []metav1.OwnerReference) (*v2v1alpha1.Jenkins, error) {
	plog.V(2).Info("start getting jenkins owner", "owner name", name)
	if ow := GetOwnerReference(consts.JenkinsKind, ors); ow != nil {
		plog.V(2).Info("trying to fetch jenkins owner from reference", "jenkins name", ow.Name)
		return GetJenkinsInstance(c, ow.Name, namespace)
	}
	if ownerName != nil {
		log.Info("trying to fetch jenkins owner from spec", "jenkins name", ownerName)
		return GetJenkinsInstance(c, *ownerName, namespace)
	}
	plog.V(2).Info("trying to fetch first jenkins instance", "namespace", namespace)
	j, err := GetFirstJenkinsInstance(c, namespace)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func GetJenkinsInstance(c client.Client, name, namespace string) (*v2v1alpha1.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &v2v1alpha1.Jenkins{}
	if err := c.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get jenkins instance by name %v", name)
	}
	return instance, nil
}

func GetFirstJenkinsInstance(c client.Client, namespace string) (*v2v1alpha1.Jenkins, error) {
	list := &v2v1alpha1.JenkinsList{}
	if err := c.List(context.TODO(), &client.ListOptions{Namespace: namespace}, list); err != nil {
		return nil, errors.Wrapf(err, "couldn't get Jenkins instances in namespace %v", namespace)
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("at least one Jenkins instance should be accessible")
	}
	j := list.Items[0]
	return GetJenkinsInstance(c, j.Name, j.Namespace)
}

func GetJenkinsFolderInstance(c client.Client, name, namespace string) (*v2v1alpha1.JenkinsFolder, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &v2v1alpha1.JenkinsFolder{}
	if err := c.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

// SetControllerReference sets owner as a Controller OwnerReference on owned.
// This is used for garbage collection of the owned object and for
// reconciling the owner object on changes to owned (with a Watch + EnqueueRequestForOwner).
// Since only one OwnerReference can be a controller, it returns an error if
// there is another OwnerReference with Controller flag set.
func SetControllerReference(owner, object v1.Object, scheme *runtime.Scheme, isController bool) error {
	ro, ok := owner.(runtime.Object)
	if !ok {
		return fmt.Errorf("is not a %T a runtime.Object, cannot call SetControllerReference", owner)
	}

	gvk, err := apiutil.GVKForObject(ro, scheme)
	if err != nil {
		return err
	}

	// Create a new ref
	ref := *newControllerRef(owner, schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind}, isController)

	existingRefs := object.GetOwnerReferences()
	fi := -1
	for i, r := range existingRefs {
		if referSameObject(ref, r) {
			fi = i
		} else if r.Controller != nil && *r.Controller {
			return newAlreadyOwnedError(object, r)
		}
	}
	if fi == -1 {
		existingRefs = append(existingRefs, ref)
	} else {
		existingRefs[fi] = ref
	}

	// Update owner references
	object.SetOwnerReferences(existingRefs)
	return nil
}

func newControllerRef(owner v1.Object, gvk schema.GroupVersionKind, isController bool) *v1.OwnerReference {
	blockOwnerDeletion := true
	or := &v1.OwnerReference{
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

// Returns true if a and b point to the same object
func referSameObject(a, b v1.OwnerReference) bool {
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

func newAlreadyOwnedError(Object v1.Object, Owner v1.OwnerReference) *controllerutil.AlreadyOwnedError {
	return &controllerutil.AlreadyOwnedError{
		Object: Object,
		Owner:  Owner,
	}
}
