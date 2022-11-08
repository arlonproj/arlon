package appprofile

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	argoappapi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	arlonapp "github.com/arlonproj/arlon/pkg/app"
	arlonclusters "github.com/arlonproj/arlon/pkg/cluster"
	"github.com/arlonproj/arlon/pkg/common"
	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

var (
	mtx sync.Mutex
)

func Reconcile(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	req ctrl.Request,
	log logr.Logger,
) (ctrl.Result, error) {
	log.Info("reconciling arlon appprofile")
	var prof arlonv1.AppProfile

	if err := cli.Get(ctx, req.NamespacedName, &prof); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("appprofile is gone -- ok")
		} else {
			log.Info(fmt.Sprintf("unable to get appprofile (%s) ... requeuing", err))
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ReconcileEverything(ctx, cli, argocli, log)
}

func ReconcileEverything(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	log logr.Logger,
) (ctrl.Result, error) {
	mtx.Lock()
	defer mtx.Unlock()
	log.V(1).Info("--- global reconciliation begin ---")
	// Get ArgoCD clusters
	conn, clApi, err := argocli.NewClusterClient()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get argocd clusters client: %s", err)
	}
	defer conn.Close()
	argoClusters, err := clApi.List(ctx, &clusterpkg.ClusterQuery{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list argocd clusters client: %s", err)
	}

	// Get arlon clusters (argocd applications)
	conn, appApi, err := argocli.NewApplicationClient()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get argocd clusters client: %s", err)
	}
	query := arlonclusters.ArlonGen2ClusterLabelQueryOnArgoApps
	arlonClusters, err := appApi.List(ctx, &argoapp.ApplicationQuery{Selector: &query})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list argocd applications: %s", err)
	}
	arlonClusterMap := make(map[string]argoappapi.Application)
	for _, arlonClust := range arlonClusters.Items {
		arlonClusterMap[arlonClust.Name] = arlonClust
	}

	// Get applications (applicationsets managed by Arlon)
	var appList appset.ApplicationSetList
	rqmt, err := labels.NewRequirement("arlon-type", selection.In, []string{"application"})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create requirement: %s", err)
	}
	sel := labels.NewSelector().Add(*rqmt)
	err = cli.List(ctx, &appList, &client.ListOptions{
		Namespace:     "argocd",
		LabelSelector: sel,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list applicationsets: %s", err)
	}
	appToProfiles := make(map[string][]string)
	validAppNames := sets.NewSet[string]()
	for _, item := range appList.Items {
		validAppNames.Add(item.Name)
		// appToProfiles[item.Name] = []string{}
	}
	log.V(1).Info("apps counted", "count", len(appList.Items))

	// Get profiles
	var profList arlonv1.AppProfileList
	err = cli.List(ctx, &profList)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list app profiles: %s", err)
	}
	log.V(1).Info("app profiles counted", "count", len(profList.Items))

	// Reconcile profiles
	profNames := sets.NewSet[string]()
	for _, prof := range profList.Items {
		profNames.Add(prof.Name)
		dirty := false
		beforeInvalidNames := sets.NewSet[string](prof.Status.InvalidAppNames...)
		afterInvalidNames := sets.NewSet[string]()
		for _, appName := range prof.Spec.AppNames {
			if !validAppNames.Contains(appName) {
				afterInvalidNames.Add(appName)
			} else {
				// Add this profile to the app's profile list
				appToProfiles[appName] = append(appToProfiles[appName], prof.Name)
			}
		}
		if !beforeInvalidNames.Equal(afterInvalidNames) {
			prof.Status.InvalidAppNames = afterInvalidNames.ToSlice()
			dirty = true
		}
		beforeHealth := prof.Status.Health
		var afterHealth string
		if len(prof.Status.InvalidAppNames) > 0 {
			afterHealth = "degraded"
		} else {
			afterHealth = "healthy"
		}
		if beforeHealth != afterHealth {
			prof.Status.Health = afterHealth
			dirty = true
		}
		if dirty {
			// update profile status
			log.Info("updating app profile", "profileName", prof.Name)
			err = cli.Status().Update(ctx, &prof)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update app profile: %s", err)
			}
		}
	}

	// Reconcile clusters
	for _, argoClust := range argoClusters.Items {
		if argoClust.Labels == nil {
			argoClust.Labels = make(map[string]string)
		}
		argoClustLabel := argoClust.Labels[arlonapp.ProfileLabelKey]
		dirty := false
		arlonClust, ok := arlonClusterMap[argoClust.Name]
		if !ok {
			// No corresponding arlon cluster. Could be an "external" cluster,
			// so allow the label to be managed independently.
			log.V(1).Info("argo cluster has no corresponding arlon cluster, skipping",
				"argoClusterName", argoClust.Name)
			continue
		}
		// Arlon cluster exists. Ensure argocd cluster is labeled identically
		if arlonClust.Labels == nil {
			arlonClust.Labels = make(map[string]string)
		}
		arlonClustLabel := arlonClust.Labels[arlonapp.ProfileLabelKey]
		if arlonClustLabel != "" {
			// Arlon cluster has label
			if argoClustLabel != arlonClustLabel {
				log.Info("updating label on argo cluster to match arlon cluster's",
					"clustName", arlonClust.Name,
					"labelValue", arlonClustLabel)
				argoClust.Labels[arlonapp.ProfileLabelKey] = arlonClustLabel
				dirty = true
			} else {
				log.V(1).Info("argo and arlon clusters already in sync",
					"clustName", arlonClust.Name)
			}
		} else if argoClustLabel != "" {
			// Arlon cluster has no label but argo cluster has one
			log.Info("removing label from argo cluster because arlon cluster has none",
				"argoClusterName", argoClust.Name)
			delete(argoClust.Labels, arlonapp.ProfileLabelKey)
			dirty = true
		} else {
			log.V(1).Info("argo & arlon cluster have no label, skipping",
				"clustName", arlonClust.Name)
		}
		if dirty {
			_, err = clApi.Update(context.Background(), &clusterpkg.ClusterUpdateRequest{
				Cluster:       &argoClust,
				UpdatedFields: []string{"labels"},
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update argo cluster: %s", err)
			}
		}
	}

	// Reconcile apps
	for _, app := range appList.Items {
		if len(app.Spec.Generators) != 1 {
			log.Info("invalid application set, has no generators",
				"appSetName", app.Name)
			continue
		}
		clustGen := app.Spec.Generators[0].Clusters
		if clustGen == nil {
			log.Info("invalid application set, generator is not of type 'clusters'",
				"appSetName", app.Name)
			continue
		}
		if len(clustGen.Selector.MatchExpressions) != 1 {
			log.Info("invalid application set, unexpected number of matchExpressions",
				"numMatchExpr", len(clustGen.Selector.MatchExpressions))
			continue
		}
		matchExpr := clustGen.Selector.MatchExpressions[0]
		if matchExpr.Key != common.ProfileAnnotationKey || matchExpr.Operator != metav1.LabelSelectorOpIn {
			log.Info("invalid application set, unexpected key or operator",
				"key", matchExpr.Key, "operator", matchExpr.Operator)
			continue
		}
		beforeLabelValues := sets.NewSet[string](matchExpr.Values...)
		afterLabelValues := sets.NewSet[string](appToProfiles[app.Name]...)
		if beforeLabelValues.Equal(afterLabelValues) {
			continue
		}
		log.Info("updating app's matching expression values",
			"app", app.Name, "values", appToProfiles[app.Name])
		clustGen.Selector.MatchExpressions[0].Values = appToProfiles[app.Name]
		err = cli.Update(ctx, &app)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update app: %s", err)
		}
	}
	log.V(1).Info("--- global reconciliation end ---")
	return ctrl.Result{}, nil
}
