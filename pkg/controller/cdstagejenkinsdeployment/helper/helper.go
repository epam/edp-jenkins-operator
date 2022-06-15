package helper

import (
	"context"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCDStageDeploy(client client.Client, name, ns string) (*codebaseApi.CDStageDeploy, error) {
	i := &codebaseApi.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	if err := client.Get(context.TODO(), nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
