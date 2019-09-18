package helper

import (
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"os"
	"path/filepath"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("controller_jenkins")

func GetExecutableFilePath() string {
	executableFilePath, err := os.Executable()
	if err != nil {
		log.Error(err, "Couldn't get executable path")
	}
	return filepath.Dir(executableFilePath)
}

func GenerateAnnotationKey(entitySuffix string) string {
	key := fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
	return key
}
