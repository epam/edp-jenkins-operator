package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsJobBuildRun_GetDeleteAfterCompletionInterval_Empty(t *testing.T) {
	instance := JenkinsJobBuildRun{}
	timeInterval := instance.GetDeleteAfterCompletionInterval()
	assert.Equal(t, time.Hour, timeInterval)
}

func TestJenkinsJobBuildRun_GetDeleteAfterCompletionInterval(t *testing.T) {
	str := "2h"
	instance := JenkinsJobBuildRun{Spec: JenkinsJobBuildRunSpec{DeleteAfterCompletionInterval: &str}}
	timeInterval := instance.GetDeleteAfterCompletionInterval()
	assert.Equal(t, 2*time.Hour, timeInterval)
}
