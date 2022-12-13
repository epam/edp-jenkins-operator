package helper

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
)

func GetCDStageDeploy(k8sClient client.Client, name, namespace string) (*codebaseApi.CDStageDeploy, error) {
	cdStageDeploy := &codebaseApi.CDStageDeploy{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := k8sClient.Get(context.TODO(), namespacedName, cdStageDeploy); err != nil {
		return nil, fmt.Errorf("failed to get CD Stage Deploy: %w", err)
	}

	return cdStageDeploy, nil
}
