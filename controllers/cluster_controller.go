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
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/io"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	corev1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/argocd"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/go-logr/logr"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var retryDelayAsResult = ctrl.Result{RequeueAfter: time.Second * 10}

// Default git location of Helm chart for Arlon app (for a cluster)
var defaultArlonChart = arlonv1.RepoSpec{
	Url:      "https://github.com/arlonproj/arlon.git",
	Path:     "pkg/cluster/manifests",
	Revision: "v0.10.0",
}

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ArgocdClient apiclient.Client
	Config       *restclient.Config
	ArgoCdNs     string
	ArlonNs      string
}

//+kubebuilder:rbac:groups=core.arlon.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.arlon.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.arlon.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("arlon Cluster")
	var cr arlonv1.Cluster

	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("cluster is gone -- ok")
			return ctrl.Result{}, nil
		}
		log.Info(fmt.Sprintf("unable to get cluster (%s) ... requeuing", err))
		return ctrl.Result{Requeue: true}, nil
	}
	// Initialize the patch helper. It stores a "before" copy of the current object.
	patchHelper, err := patch.NewHelper(&cr, r.Client)
	if err != nil {
		log.Error(err, "Failed to configure the patch helper")
		return ctrl.Result{Requeue: true}, nil
	}

	conn, appIf, err := r.ArgocdClient.NewApplicationClient()
	if err != nil {
		msg := fmt.Sprintf("failed to get argocd application client: %s", err)
		return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
	}
	defer io.Close(conn)

	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		// Handle deletion reconciliation loop.
		return r.reconcileDelete(ctx, log, &cr, patchHelper, appIf)
	}
	if cr.Status.State == "created" {
		log.V(1).Info("Cluster is already created")
		return ctrl.Result{}, nil
	}
	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(&cr, arlonv1.ClusterFinalizer) {
		controllerutil.AddFinalizer(&cr, arlonv1.ClusterFinalizer)
		// patch and return right away instead of reusing the main defer,
		// because the main defer may take too much time to get cluster status
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
		if err := patchHelper.Patch(ctx, &cr, patchOpts...); err != nil {
			log.Error(err, "Failed to patch cluster to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	ctmpl := &cr.Spec.ClusterTemplate
	repoUrl := ctmpl.Url
	repoRevision := ctmpl.Revision
	repoPath := ctmpl.Path
	if cr.Status.InnerClusterName == "" {
		log.Info("validating cluster template ...")
		_, creds, err := argocd.GetKubeclientAndRepoCreds(r.Config, r.ArgoCdNs,
			repoUrl)
		if err != nil {
			msg := fmt.Sprintf("failed to get repo creds: %s", err)
			return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
		}
		innerClusterName, err := bcl.ValidateGitDir(creds, repoUrl, repoRevision, repoPath)
		if err != nil {
			msg := fmt.Sprintf("failed to validate cluster template: %s", err)
			return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
		}
		cr.Status.InnerClusterName = innerClusterName
		return r.UpdateState(ctx, log, &cr, "template-validated",
			"cluster template validation successful", ctrl.Result{})
	}

	ovr := cr.Spec.Override
	overridden := ovr != nil
	if overridden {
		if !cr.Status.OverrideSuccessful {
			// Handle override
			err = cluster.CreatePatchDir(r.Config, cr.Name, ovr.Repo.Url, r.ArgoCdNs,
				ovr.Repo.Path, ovr.Repo.Revision,
				repoRevision, []byte(ovr.Patch), repoUrl, repoPath)
			if err != nil {
				msg := fmt.Sprintf("failed to create override patch in git: %s", err)
				return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
			}
			cr.Status.OverrideSuccessful = true
			return r.UpdateState(ctx, log, &cr, "override-created",
				"override patch creation successful", ctrl.Result{})
		}
		// Point the cluster to the override instead of cluster template
		repoUrl = ovr.Repo.Url
		repoRevision = ovr.Repo.Revision
		repoPath = ovr.Repo.Path
	}

	// Check if arlon app already exists
	aan := arlonAppName(cr.Name)
	_, err = appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &aan})
	if err != nil {
		grpcStatus, ok := grpcstatus.FromError(err)
		if !ok {
			return r.UpdateState(ctx, log, &cr, "retrying",
				"failed to get grpc status from argocd API", retryDelayAsResult)
		}
		if grpcStatus.Code() != grpccodes.NotFound {
			return r.UpdateState(ctx, log, &cr, "retrying",
				fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
				retryDelayAsResult)
		}
		casMgmtClusterHost := ""
		innerClusterName := cr.Status.InnerClusterName
		gen2CASEnabled := cr.Spec.Autoscaler != nil
		if gen2CASEnabled {
			casMgmtClusterHost = cr.Spec.Autoscaler.MgmtClusterHost
		}
		arlonHelmChart := cr.Spec.ArlonHelmChart
		if arlonHelmChart == nil {
			arlonHelmChart = &defaultArlonChart
		}
		_, err = cluster.Create(appIf, r.Config, r.ArgoCdNs, r.ArlonNs,
			cr.Name, innerClusterName, arlonHelmChart.Url, arlonHelmChart.Revision,
			arlonHelmChart.Path, "",
			nil, true, casMgmtClusterHost, gen2CASEnabled)
		if err != nil {
			msg := fmt.Sprintf("failed to create arlon application: %s", err)
			return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
		}
	}
	// Check if cluster app already exists
	_, err = appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &cr.Name})
	if err == nil {
		// We're done
		return r.UpdateState(ctx, log, &cr, "created",
			"cluster app already exists -- ok", ctrl.Result{})
	}
	// FIXME: I think url, revision and path are wrong if overridden
	_, err = cluster.CreateClusterApp(appIf, r.ArgoCdNs,
		cr.Name, cr.Status.InnerClusterName, repoUrl, repoRevision,
		repoPath, true, overridden)
	if err != nil {
		msg := fmt.Sprintf("failed to create cluster application: %s", err)
		return r.UpdateState(ctx, log, &cr, "retrying", msg, retryDelayAsResult)
	}
	return r.UpdateState(ctx, log, &cr, "created",
		"cluster creation successful", ctrl.Result{})
}

func (r *ClusterReconciler) UpdateState(
	ctx context.Context,
	log logr.Logger,
	cr *arlonv1.Cluster,
	state string,
	msg string,
	result ctrl.Result,
) (ctrl.Result, error) {
	cr.Status.State = state
	cr.Status.Message = msg
	log.Info(fmt.Sprintf("%s ... setting state to '%s'", msg, cr.Status.State))
	if err := r.Status().Update(ctx, cr); err != nil {
		log.Error(err, "unable to update clusterregistration status")
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *ClusterReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	cr *arlonv1.Cluster,
	patchHelper *patch.Helper,
	appIf argoapp.ApplicationServiceClient,
) (ctrl.Result, error) {
	// Check if cluster app exists
	clusterApp, err := appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &cr.Name})
	if err == nil {
		if !clusterApp.DeletionTimestamp.IsZero() {
			log.Info("cluster app deletion already pending -- will check again later")
			return retryDelayAsResult, nil
		}
		// Delete it
		cascade := true
		_, err = appIf.Delete(ctx, &argoapp.ApplicationDeleteRequest{
			Name:    &cr.Name,
			Cascade: &cascade,
		})
		if err != nil {
			msg := fmt.Sprintf("failed to delete cluster app: %s", err)
			return r.UpdateState(ctx, log, cr, "error-deleting-cluster-app",
				msg, retryDelayAsResult)
		}
		return r.UpdateState(ctx, log, cr, "deleting-cluster-app",
			"deleting cluster app", ctrl.Result{})
	}
	grpcStatus, ok := grpcstatus.FromError(err)
	if !ok {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			"failed to get grpc status from argocd API", retryDelayAsResult)
	}
	if grpcStatus.Code() != grpccodes.NotFound {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
			retryDelayAsResult)
	}

	// Check if arlon app already exists
	aan := arlonAppName(cr.Name)
	arlonApp, err := appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &aan})
	if err == nil {
		if !arlonApp.DeletionTimestamp.IsZero() {
			log.Info("arlon app deletion already pending -- will check again later")
			return retryDelayAsResult, nil
		}
		// Delete it
		cascade := true
		_, err = appIf.Delete(ctx, &argoapp.ApplicationDeleteRequest{
			Name:    &aan,
			Cascade: &cascade,
		})
		if err != nil {
			msg := fmt.Sprintf("failed to delete arlon app: %s", err)
			return r.UpdateState(ctx, log, cr, "error-deleting-arlon-app",
				msg, retryDelayAsResult)
		}
		return r.UpdateState(ctx, log, cr, "deleting-arlon-app",
			"deleting arlon app", ctrl.Result{})
	}
	grpcStatus, ok = grpcstatus.FromError(err)
	if !ok {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			"failed to get grpc status from argocd API", retryDelayAsResult)
	}
	if grpcStatus.Code() != grpccodes.NotFound {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
			retryDelayAsResult)
	}
	controllerutil.RemoveFinalizer(cr, arlonv1.ClusterFinalizer)
	if err := patchHelper.Patch(ctx, cr); err != nil {
		log.Info(fmt.Sprintf("failed to remove finalizer from cluster: %s", err))
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("removed finalizer from cluster '%s'",
		cr.Name))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Cluster{}).
		Complete(r)
}

func arlonAppName(clusterName string) string {
	return fmt.Sprintf("%s-arlon", clusterName)
}
