package helper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePath(t *testing.T) {
	res, err := createPath("test", false)
	assert.NoError(t, err)
	assert.Equal(t, res, fmt.Sprintf("%s/%s", DefaultConfigsAbsolutePath, "test"))
}
