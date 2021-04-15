package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/triggerjenkinsdeployjob"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDefChain(client client.Client, service ps.PlatformService) handler.CDStageJenkinsDeploymentHandler {
	return triggerjenkinsdeployjob.TriggerJenkinsDeployJob{
		Client:   client,
		Platform: service,
		Log:      ctrl.Log.WithName("cd-stage-jenkins-deployment-chain"),
	}
}
