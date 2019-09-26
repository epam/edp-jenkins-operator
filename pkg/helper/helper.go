package helper

import (
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"os"
	"path/filepath"
)


func GetExecutableFilePath() (string, error) {
	executableFilePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executableFilePath), nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}


func GenerateAnnotationKey(entitySuffix string) string {
	key := fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
	return key
}
