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
	arlov1 "arlo.org/arlo/api/v1"
	"arlo.org/arlo/pkg/argocd"
	"context"
	"fmt"
	cmdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/util/clusterauth"
	"github.com/argoproj/argo-cd/v2/util/io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var argocdclient apiclient.Client

// ClusterRegistrationReconciler reconciles a ClusterRegistration object
type ClusterRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arlo.arlo.org,resources=clusterregistrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arlo.arlo.org,resources=clusterregistrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arlo.arlo.org,resources=clusterregistrations/finalizers,verbs=update

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
	var cr arlov1.ClusterRegistration
	// your logic here
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		log.Error(err, "unable to get clusterregistration")
		return ctrl.Result{}, err
	}
	if cr.Status.State == "complete" {
		log.V(1).Info("clusterregistration is already complete")
		return ctrl.Result{}, nil
	}
	if cr.Status.State == "error" {
		log.V(1).Info("clusterregistration is already in error state")
		return ctrl.Result{}, nil
	}
	if cr.Spec.ApiEndpoint == "" || cr.Spec.KubeconfigSecretName == "" {
		msg := "clusterregistration has an invalid spec"
		log.V(0).Info(msg)
		cr.Status.State = "error"
		cr.Status.Message = msg
		if err := r.Status().Update(ctx, &cr); err != nil {
			log.Error(err, "unable to update clusterregistration status")
			return ctrl.Result{}, err
		}
		log.Info("set status to error")
		return ctrl.Result{}, nil
		//return ctrl.Result{RequeueAfter: 60*time.Second}, nil
	}
	conn, clusterIf := argocdclient.NewClusterClientOrDie()
	defer io.Close(conn)
	clquery := cluster.ClusterQuery{Name: cr.Spec.ClusterName}
	clust, err := clusterIf.Get(ctx, &clquery)
	if err == nil {
		if clust.Server == cr.Spec.ApiEndpoint {
			log.Info(fmt.Sprintf("cluster %s already exists -- ok", cr.Spec.ClusterName))
			cr.Status.State = "complete"
		} else {
			log.Info("cluster already exists but its API endpoint does not match")
			cr.Status.State = "error"
		}
		if err := r.Status().Update(ctx, &cr); err != nil {
			log.Error(err, "unable to update clusterregistration status")
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("successfully set status to %s", cr.Status.State))
		return ctrl.Result{}, nil
	}
	log.Info(fmt.Sprintf("failed to lookup existing cluster -- this is expected if new: %s", err))
	var secret corev1.Secret
	secretNamespacedName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      cr.Spec.KubeconfigSecretName,
	}
	if err := r.Get(ctx, secretNamespacedName, &secret); err != nil {
		msg := fmt.Sprintf("failed to read secret: %s", err.Error())
		cr.Status.State = "error"
		cr.Status.Message = msg
		log.Info(msg)
		if err := r.Status().Update(ctx, &cr); err != nil {
			log.Error(err, "unable to update clusterregistration status")
			return ctrl.Result{}, err
		}
		log.Info("set status to error")
		return ctrl.Result{}, nil
	}
	conf, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["kubeconfig"])
	if err != nil {
		log.Error(err, "failed to read kubeconfig from secret")
		return ctrl.Result{}, err
	}
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Error(err, "failed to get clientset from config")
		return ctrl.Result{}, err
	}
	cr.Status.State = "complete"
	if err := r.Status().Update(ctx, &cr); err != nil {
		log.Error(err, "unable to update clusterregistration status")
		return ctrl.Result{}, err
	}
	managerBearerToken, err := clusterauth.GetServiceAccountBearerToken(clientset,
		"kube-system", clusterauth.ArgoCDManagerServiceAccount)
	if err != nil {
		msg := "failed to install service account in destination cluster"
		log.Info(msg, "err", err)
		cr.Status.State = "error"
		log.Info(msg)
		if err := r.Status().Update(ctx, &cr); err != nil {
			log.Error(err, "unable to update clusterregistration status")
			return ctrl.Result{}, err
		}
		log.Info("set status to error")
		return ctrl.Result{}, nil
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
		conf,
		managerBearerToken,
		nil, // awsAuthConf
		nil, // execProviderConf
	)
	clstCreateReq := clusterpkg.ClusterCreateRequest{
		Cluster: clst,
		Upsert:  true,
	}
	_, err = clusterIf.Create(context.Background(), &clstCreateReq)
	if err != nil {
		log.Info("failed to add cluster to argocd", "msg", err.Error())
		cr.Status.State = "retrying"
		if err := r.Status().Update(ctx, &cr); err != nil {
			log.Error(err, "unable to update clusterregistration status")
			return ctrl.Result{}, err
		}
		log.Info("set status to retrying")
		return ctrl.Result{RequeueAfter: time.Second * 60}, nil
	}
	log.Info("setting status to complete")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arlov1.ClusterRegistration{}).
		Complete(r)
}

func init() {
	argocdclient = argocd.NewArgocdClientOrDie()
}
