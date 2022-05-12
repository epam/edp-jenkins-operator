package v1

import (
	"github.com/bmizerany/assert"
	"testing"
)

const (
	minMaxIncorrectValue = 4
	maxMinIncorrectValue = 7201
	correctValue         = 100
)

func TestIsAutoTriggerEnabledMethod_AutoTriggerShouldBeDisabledAsFieldIsNil(t *testing.T) {
	jj := JenkinsJob{
		Spec: JenkinsJobSpec{
			Job: Job{
				AutoTriggerPeriod: nil,
			},
		}}
	assert.Equal(t, false, jj.IsAutoTriggerEnabled())
}

func TestIsAutoTriggerEnabledMethod_AutoTriggerShouldBeDisabledAsFieldIsZero(t *testing.T) {
	jj := JenkinsJob{
		Spec: JenkinsJobSpec{
			Job: Job{
				AutoTriggerPeriod: GetInt32P(0),
			},
		}}
	assert.Equal(t, false, jj.IsAutoTriggerEnabled())
}

func GetInt32P(v int32) *int32 {
	return &v
}

func TestIsAutoTriggerEnabledMethod_AutoTriggerShouldBeDisabledAsFieldIsIncorrect(t *testing.T) {
	jj := JenkinsJob{
		Spec: JenkinsJobSpec{
			Job: Job{
				AutoTriggerPeriod: GetInt32P(minMaxIncorrectValue),
			},
		}}
	assert.Equal(t, false, jj.IsAutoTriggerEnabled())

	jj.Spec.Job.AutoTriggerPeriod = GetInt32P(maxMinIncorrectValue)
	assert.Equal(t, false, jj.IsAutoTriggerEnabled())
}

func TestIsAutoTriggerEnabledMethod_AutoTriggerShouldBeEnabled(t *testing.T) {
	jj := JenkinsJob{
		Spec: JenkinsJobSpec{
			Job: Job{
				AutoTriggerPeriod: GetInt32P(correctValue),
			},
		}}
	assert.Equal(t, true, jj.IsAutoTriggerEnabled())
}
