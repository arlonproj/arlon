package log

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetLogger() logr.Logger {
	return ctrl.Log
}
