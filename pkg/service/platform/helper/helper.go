package helper

import (
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
)

// GenerateLabels returns map with labels for k8s objects
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func RuntimeObjectsEqual(new runtime.Object, old runtime.Object) bool {
		if reflect.DeepEqual(new, old) {
			return true
		}
	return false
}