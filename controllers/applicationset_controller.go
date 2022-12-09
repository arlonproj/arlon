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
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"github.com/arlonproj/arlon/pkg/appprofile"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationSetReconciler reconciles a AppProfile object
type ApplicationSetReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ArgocdClient apiclient.Client
}

//+kubebuilder:rbac:groups=core.arlon.io,resources=appprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.arlon.io,resources=appprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.arlon.io,resources=appprofiles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppProfile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ApplicationSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here

	return reconcileApplicationSet(ctx, r.Client, r.ArgocdClient, req, logger)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appset.ApplicationSet{}).
		Complete(r)
}

func reconcileApplicationSet(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	req ctrl.Request,
	log logr.Logger,
) (ctrl.Result, error) {
	log.Info("reconciling applicationset (possible arlon app)")
	var app appset.ApplicationSet

	if err := cli.Get(ctx, req.NamespacedName, &app); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("applicationset is gone -- ok")
			// reconcile everything because app's deletion may affect profiles
			return appprofile.ReconcileEverything(ctx, cli, argocli, log)
		} else {
			log.Info(fmt.Sprintf("unable to get applicationset (%s) ... requeuing", err))
			return ctrl.Result{Requeue: true}, nil
		}
	}
	if app.Labels == nil || app.Labels["arlon-type"] != "application" {
		log.V(1).Info("applicationset is not an arlon app, skipping...")
		return ctrl.Result{}, nil
	}
	return appprofile.ReconcileEverything(ctx, cli, argocli, log)
}
