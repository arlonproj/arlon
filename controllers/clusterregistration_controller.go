/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	cmdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/util/clusterauth"
	"github.com/argoproj/argo-cd/v2/util/io"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// ClusterRegistrationReconciler reconciles a ClusterRegistration object
type ClusterRegistrationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ArgocdClient apiclient.Client
}

//+kubebuilder:rbac:groups=core.arlon.io,resources=clusterregistrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.arlon.io,resources=clusterregistrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.arlon.io,resources=clusterregistrations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterRegistration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("clusterregistration", req.NamespacedName)
	log.V(1).Info("arlo clusterregistration")
	var cr arlonv1.ClusterRegistration

	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("clusterregistration is gone -- ok")
			return ctrl.Result{}, nil
		}
		log.Info(fmt.Sprintf("unable to get clusterregistration (%s) ... requeuing", err))
		return ctrl.Result{Requeue: true}, nil
	}
	// Initialize the patch helper. It stores a "before" copy of the current object.
	patchHelper, err := patch.NewHelper(&cr, r.Client)
	if err != nil {
		log.Error(err, "Failed to configure the patch helper")
		return ctrl.Result{Requeue: true}, nil
	}
	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		// Handle deletion reconciliation loop.
		return reconcileDelete(r.ArgocdClient, ctx, log, &cr, patchHelper)
	}
	if cr.Status.State == "complete" {
		log.V(1).Info("clusterregistration is already complete")
		return ctrl.Result{}, nil
	}
	if cr.Status.State == "error" {
		log.V(1).Info("clusterregistration is already in error state")
		return ctrl.Result{}, nil
	}
	if cr.Spec.KubeconfigSecretName == "" || cr.Spec.KubeconfigSecretKeyName == "" {
		return updateState(r, log, &cr, "error", "clusterregistration has an invalid spec", ctrl.Result{})
	}
	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(&cr, arlonv1.ClusterRegistrationFinalizer) {
		controllerutil.AddFinalizer(&cr, arlonv1.ClusterRegistrationFinalizer)
		// patch and return right away instead of reusing the main defer,
		// because the main defer may take too much time to get cluster status
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
		if err := patchHelper.Patch(ctx, &cr, patchOpts...); err != nil {
			log.Error(err, "Failed to patch ClusterRegistration to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	conn, clusterIf := r.ArgocdClient.NewClusterClientOrDie()
	defer io.Close(conn)
	clquery := cluster.ClusterQuery{Name: cr.Spec.ClusterName}

	_, err = clusterIf.Get(ctx, &clquery)
	if err == nil {
		msg := fmt.Sprintf("cluster %s already exists -- ok", cr.Spec.ClusterName)
		return updateState(r, log, &cr, "complete", msg, ctrl.Result{})
	}
	log.Info(fmt.Sprintf("failed to lookup existing cluster %s -- this is expected if new: %s", cr.Spec.ClusterName, err))

	var secret corev1.Secret
	secretNamespacedName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      cr.Spec.KubeconfigSecretName,
	}
	if err := r.Get(ctx, secretNamespacedName, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			msg := fmt.Sprintf("kubeconfig secret %s does not exist yet, retrying in 10 seconds",
				cr.Spec.KubeconfigSecretName)
			return updateState(r, log, &cr, "retrying", msg, ctrl.Result{RequeueAfter: time.Second * 10})
		}
		msg := fmt.Sprintf("failed to read secret: %s", err)
		return updateState(r, log, &cr, "error", msg, ctrl.Result{})
	}
	data := secret.Data[cr.Spec.KubeconfigSecretKeyName]
	if data == nil {
		return updateState(r, log, &cr, "error",
			fmt.Sprintf("secret subkey %s does not exist",
				cr.Spec.KubeconfigSecretKeyName), ctrl.Result{})
	}
	conf, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return updateState(r, log, &cr, "error",
			fmt.Sprintf("failed to read kubeconfig from secret: %s", err),
			ctrl.Result{})
	}
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return updateState(r, log, &cr, "error",
			fmt.Sprintf("failed to get clientset from config: %s", err),
			ctrl.Result{})
	}
	managerBearerToken, err := clusterauth.InstallClusterManagerRBAC(clientset,
		"kube-system", []string{})
	if err != nil {
		return updateState(r, log, &cr, "retrying",
			fmt.Sprintf("failed to install service account in destination cluster: '%s' ... retrying in 10 secs", err),
			ctrl.Result{RequeueAfter: time.Second * 10})
	}
	log.Info("adding cluster")
	var namespaces []string
	clusterName := cr.Spec.ClusterName
	if clusterName == "" {
		clusterName = cr.Name
	}
	clst := cmdutil.NewCluster(
		clusterName,
		namespaces,
		false,
		conf,
		managerBearerToken,
		nil, // awsAuthConf
		nil, // execProviderConf
		nil, // labels
		nil, // annotations
	)
	clstCreateReq := clusterpkg.ClusterCreateRequest{
		Cluster: clst,
		Upsert:  true,
	}
	_, err = clusterIf.Create(context.Background(), &clstCreateReq)
	if err != nil {
		return updateState(r, log, &cr, "retrying",
			fmt.Sprintf("failed to add cluster to argocd: '%s' ... retrying in 10 secs", err),
			ctrl.Result{RequeueAfter: time.Second * 10})
	}
	return updateState(r, log, &cr, "complete", "successfully added cluster to argocd", ctrl.Result{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arlonv1.ClusterRegistration{}).
		Complete(r)
}

func updateState(
	r *ClusterRegistrationReconciler,
	log logr.Logger,
	cr *arlonv1.ClusterRegistration,
	state string,
	msg string,
	result ctrl.Result,
) (ctrl.Result, error) {
	cr.Status.State = state
	cr.Status.Message = msg
	log.Info(fmt.Sprintf("%s ... setting state to '%s'", msg, cr.Status.State))
	if err := r.Status().Update(context.Background(), cr); err != nil {
		log.Error(err, "unable to update clusterregistration status")
		return ctrl.Result{}, err
	}
	return result, nil
}

func reconcileDelete(
	argocdclient apiclient.Client,
	ctx context.Context,
	log logr.Logger,
	cr *arlonv1.ClusterRegistration,
	patchHelper *patch.Helper,
) (ctrl.Result, error) {
	conn, clusterIf := argocdclient.NewClusterClientOrDie()
	defer io.Close(conn)
	clquery := cluster.ClusterQuery{Name: cr.Spec.ClusterName}
	log.Info(fmt.Sprintf("reconciling deletion of clusterregistration '%s' with cluster name '%s'",
		cr.Name, cr.Spec.ClusterName))
	clust, err := clusterIf.Get(ctx, &clquery)
	if err != nil {
		log.Info(fmt.Sprintf("cluster '%s' does not exist or could not be queried (%s) -- ignoring",
			cr.Spec.ClusterName, err))
	} else {
		clquery.Server = clust.Server
		if _, err := clusterIf.Delete(ctx, &clquery); err != nil {
			log.Info(fmt.Sprintf("cluster '%s' could not be deleted from argocd (%s) -- requeuing in 10 secs",
				clquery.Name, err))
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
	}
	controllerutil.RemoveFinalizer(cr, arlonv1.ClusterRegistrationFinalizer)
	if err := patchHelper.Patch(ctx, cr); err != nil {
		log.Info(fmt.Sprintf("failed to patch clusterregistration: %s", err))
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("removed finalizer from clusterregistration '%s'",
		cr.Spec.ClusterName))
	return ctrl.Result{}, nil
}
