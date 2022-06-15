package webhook

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/controller"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)


func Start(port int) error {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
	utilruntime.Must(capi.AddToScheme(scheme))

	cfg := ctrl.GetConfigOrDie()
	cli, err := controller.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to get new kubernetes client: %s", err)
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           scheme,
		Port:             port,
		LeaderElection:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to get new manager: %s", err)
	}
	wh := newWebhook(cli, scheme)
	mgr.GetWebhookServer().Register("/rewrite", wh)
	if err := mgr.AddHealthzCheck("healthz", mgr.GetWebhookServer().StartedChecker()); err != nil {
		return fmt.Errorf("failed to set up health check: %s", err)
	}
	if err := mgr.AddReadyzCheck("readyz", mgr.GetWebhookServer().StartedChecker()); err != nil {
		return fmt.Errorf("failed to set up ready check: %s", err)
	}
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("failed to start manager: %s", err)
	}
	fmt.Println("manager started")
	return nil
}

