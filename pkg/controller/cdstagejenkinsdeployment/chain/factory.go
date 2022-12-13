package chain

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/handler"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

func CreateDefChain(k8sClient client.Client, service ps.PlatformService) handler.CDStageJenkinsDeploymentHandler {
	log := ctrl.Log.WithName("cd-stage-jenkins-deployment-chain")

	return TriggerJenkinsDeployJob{
		client:   k8sClient,
		platform: service,
		log:      log,
		next: DeleteCDStageDeploy{
			client: k8sClient,
			log:    log,
		},
	}
}

func nextServeOrNil(next handler.CDStageJenkinsDeploymentHandler, jd *jenkinsApi.CDStageJenkinsDeployment) error {
	if next == nil {
		return nil
	}

	if err := next.ServeRequest(jd); err != nil {
		return fmt.Errorf("failed to perform next ServeRequest: %w", err)
	}

	return nil
}
