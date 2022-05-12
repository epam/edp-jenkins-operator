package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCDStageJenkinsDeployment_SetFailedStatus(t *testing.T) {
	instance := CDStageJenkinsDeployment{}
	errTest := errors.New("test")
	instance.SetFailedStatus(errTest)
	assert.Equal(t, errTest.Error(), instance.Status.Message)
	assert.Equal(t, failed, instance.Status.Status)
}
