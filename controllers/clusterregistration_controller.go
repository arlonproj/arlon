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
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		log.V(1).Info("clusterregistration is in error state")
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
	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		log.Error(err, "unable to get secret")
		return ctrl.Result{}, err
	}
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
