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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var plog = logf.Log.WithName("platform_util")

func GetStageInstanceOwner(c client.Client, jj v1alpha1.JenkinsJob) (*edpv1alpha1.Stage, error) {
	plog.V(2).Info("start getting stage owner cr", "stage", jj.Name)
	if ow := GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		plog.V(2).Info("trying to fetch stage owner from reference", "stage", ow.Name)
		return getStageInstance(c, ow.Name, jj.Namespace)
	}
	if jj.Spec.StageName != nil {
		plog.V(2).Info("trying to fetch stage owner from spec", "stage", jj.Spec.StageName)
		return getStageInstance(c, *jj.Spec.StageName, jj.Namespace)
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

func getStageInstance(c client.Client, name, namespace string) (*edpv1alpha1.Stage, error) {
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

func GetCDPipelineInstance(c client.Client, name, namespace string) (*edpv1alpha1.CDPipeline, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &edpv1alpha1.CDPipeline{}
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
		return getJenkinsInstance(c, ow.Name, namespace)
	}
	if ownerName != nil {
		log.Info("trying to fetch jenkins owner from spec", "jenkins name", ownerName)
		return getJenkinsInstance(c, *ownerName, namespace)
	}
	plog.V(2).Info("trying to fetch first jenkins instance", "namespace", namespace)
	j, err := getFirstJenkinsInstance(c, namespace)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func getJenkinsInstance(c client.Client, name, namespace string) (*v2v1alpha1.Jenkins, error) {
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

func getFirstJenkinsInstance(c client.Client, namespace string) (*v2v1alpha1.Jenkins, error) {
	list := &v2v1alpha1.JenkinsList{}
	if err := c.List(context.TODO(), &client.ListOptions{Namespace: namespace}, list); err != nil {
		return nil, errors.Wrapf(err, "couldn't get Jenkins instances in namespace %v", namespace)
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("at least one Jenkins instance should be accessible")
	}
	j := list.Items[0]
	return getJenkinsInstance(c, j.Name, j.Namespace)
}
