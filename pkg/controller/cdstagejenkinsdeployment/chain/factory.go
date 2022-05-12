package chain

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

func CreateDefChain(client client.Client, service ps.PlatformService) handler.CDStageJenkinsDeploymentHandler {
	log := ctrl.Log.WithName("cd-stage-jenkins-deployment-chain")
	return TriggerJenkinsDeployJob{
		client:   client,
		platform: service,
		log:      log,
		next: DeleteCDStageDeploy{
			client: client,
			log:    log,
		},
	}
}

func nextServeOrNil(next handler.CDStageJenkinsDeploymentHandler, jd *jenkinsApi.CDStageJenkinsDeployment) error {
	if next != nil {
		return next.ServeRequest(jd)
	}
	return nil
}
