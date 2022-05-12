package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToScheme(t *testing.T) {
	sch := runtime.NewScheme()
	assert.NoError(t, AddToScheme(sch))
}
