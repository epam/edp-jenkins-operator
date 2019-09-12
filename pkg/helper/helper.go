package helper

import (
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

