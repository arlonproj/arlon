package controller

import (
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/controllers"
	"github.com/arlonproj/arlon/pkg/argocd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	cmd      string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(arlonv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func NewClient(config *rest.Config) (client.Client, error) {
	return client.New(config, client.Options{
		Scheme: scheme,
	})
}

func StartController(argocdConfigPath string, metricsAddr string, probeAddr string, enableLeaderElection bool) {
	argocdClient := argocd.NewArgocdClientOrDie(argocdConfigPath)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d4242dee.arlon.io",
		// Disable caching for secret objects, because the controller reads them
		// in a particular namespace. Caching requires RBAC to be setup for
		// cluster-wide List access, as opposed to the more secure
		// namespace-scoped access.
		ClientDisableCacheFor: []client.Object{
			&corev1.Secret{},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterRegistrationReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterRegistration")
		os.Exit(1)
	}
	if err = (&controllers.CallHomeConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CallHomeConfig")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func StartCallHomeController(metricsAddr string, probeAddr string, enableLeaderElection bool) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d5252dee.arlon.io",
		// Disable caching for secret objects, because the controller reads them
		// in a particular namespace. Caching requires RBAC to be setup for
		// cluster-wide List access, as opposed to the more secure
		// namespace-scoped access.
		ClientDisableCacheFor: []client.Object{
			&corev1.Secret{},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.CallHomeConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterRegistration")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
