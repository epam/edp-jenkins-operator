package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJenkinsJobBuildRun_GetDeleteAfterCompletionInterval_Empty(t *testing.T) {
	instance := JenkinsJobBuildRun{}
	assert.Equal(t, time.Hour, instance.GetDeleteAfterCompletionInterval())
}

func TestJenkinsJobBuildRun_GetDeleteAfterCompletionInterval(t *testing.T) {
	str := "2h"
	instance := JenkinsJobBuildRun{Spec: JenkinsJobBuildRunSpec{DeleteAfterCompletionInterval: &str}}
	assert.Equal(t, 2*time.Hour, instance.GetDeleteAfterCompletionInterval())
}
