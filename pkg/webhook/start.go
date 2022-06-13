package webhook

import (
	"fmt"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)
func Start(port int) error {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		Port:             port,
		LeaderElection:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to get new manager: %s", err)
	}
	return nil
}

