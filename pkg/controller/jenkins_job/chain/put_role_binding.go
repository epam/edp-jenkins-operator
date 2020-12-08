package chain

import (
	"fmt"
	"github.com/epmd-edp/codebase-operator/v2/pkg/openshift"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	handler2 "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
)

type PutRoleBinding struct {
	next handler2.JenkinsJobHandler
	cs   openshift.ClientSet
	ps   platform.PlatformService
}

func (h PutRoleBinding) ServeRequest(jj *v1alpha1.JenkinsJob) error {
	log.V(2).Info("start creating role binding")
	if err := setIntermediateStatus(h.cs.Client, jj, v1alpha1.RoleBinding); err != nil {
		return err
	}
	if err := h.tryToCreateRoleBinding(jj); err != nil {
		if err := setFailStatus(h.cs.Client, jj, v1alpha1.RoleBinding, err.Error()); err != nil {
			return err
		}
		return err
	}
	log.V(2).Info("end creating role binding")
	return nextServeOrNil(h.next, jj)
}

func (h PutRoleBinding) tryToCreateRoleBinding(jj *v1alpha1.JenkinsJob) error {
	d, err := h.ps.GetConfigMapData(jj.Namespace, "edp-config")
	if err != nil {
		return err
	}
	s, err := plutil.GetStageInstanceOwner(h.cs.Client, *jj)
	if err != nil {
		return err
	}
	en := d["edp_name"]
	pn := fmt.Sprintf("%v-%v", en, s.Name)
	if err := h.createAdminRoleBinding(en, pn, jj.Namespace); err != nil {
		return errors.Wrap(err, "an error has occurred while creating admin role binging")
	}
	log.Info("admin role binding has been created", "roleBindingName", en + "-admin")
	if err := h.createViewRoleBinding(en, pn); err != nil {
		return errors.Wrap(err, "an error has occurred while creating view role binging")
	}
	log.Info("view role binding has been created", "roleBindingName", en + "-view")
	return nil
}

func (h PutRoleBinding) createAdminRoleBinding(edpName, projectName, namespace string) error {
	rb, err := h.ps.GetRoleBinding(edpName + "-admin", projectName)
	if err != nil {
		if k8sError.IsNotFound(err) == true {
			log.V(2).Info("admin RoleBinding not found, it will be created", "roleBindingName", edpName + "-admin")
		} else {
			return errors.Wrapf(err, "an error has been occurred while getting admin RoleBinding")
		}
	}
	if rb != nil {
		log.V(2).Info("admin RoleBinding already exist, its creation will be skipped", "roleBindingName", edpName + "-admin")
		return nil
	}
	return h.ps.CreateRoleBinding(
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "admin", APIGroup: consts.AuthorizationApiGroup, Kind: consts.ClusterRoleKind},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-super-admin"},
			{Kind: "Group", Name: edpName + "-edp-admin"},
			{Kind: "ServiceAccount", Name: consts.JenkinsServiceAccount, Namespace: namespace},
			{Kind: "ServiceAccount", Name: consts.EdpAdminConsoleServiceAccount, Namespace: namespace},
		},
	)
}

func (h PutRoleBinding) createViewRoleBinding(edpName, projectName string) error {
	rb, err := h.ps.GetRoleBinding(edpName + "-view", projectName)
	if err != nil {
		if k8sError.IsNotFound(err) == true {
			log.V(2).Info("view RoleBinding not found, it will be created", "roleBindingName", edpName + "-view")
		} else {
			return errors.Wrapf(err, "an error has been occurred while getting view RoleBinding")
		}
	}
	if rb != nil {
		log.V(2).Info("view RoleBinding already exist, its creation will be skipped", "roleBindingName", edpName + "-view")
		return nil
	}
	return h.ps.CreateRoleBinding(
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "view", APIGroup: consts.AuthorizationApiGroup, Kind: consts.ClusterRoleKind},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-view"},
		},
	)
}

