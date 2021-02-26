package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/triggerjenkinsdeployjob"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDefChain(client client.Client, service ps.PlatformService) handler.CDStageJenkinsDeploymentHandler {
	return triggerjenkinsdeployjob.TriggerJenkinsDeployJob{
		Client:   client,
		Platform: service,
	}
}
