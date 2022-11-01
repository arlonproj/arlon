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
	sets "github.com/deckarep/golang-set"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
func init() {
	conn, clusterIf := r.ArgocdClient.NewClusterClientOrDie()
	defer io.Close(conn)
}
*/

func Reconcile(
	ctx context.Context,
	cli client.Client,
	req ctrl.Request,
	argocli argoclient.Client,
	log logr.Logger,
) (ctrl.Result, error) {
	log.V(1).Info("arlon appprofile")
	var prof arlonv1.AppProfile

	if err := cli.Get(ctx, req.NamespacedName, &prof); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("appprofile is gone -- ok")
			// return ctrl.Result{}, nil
		}
		log.Info(fmt.Sprintf("unable to get appprofile (%s) ... requeuing", err))
		return ctrl.Result{Requeue: true}, nil
	}
	return reconcileEverything(ctx, cli, req, argocli, log)
}

func reconcileEverything(
	ctx context.Context,
	cli client.Client,
	req ctrl.Request,
	argocli argoclient.Client,
	log logr.Logger,
) (ctrl.Result, error) {

	// Get ArgoCD clusters
	conn, clApi, err := argocli.NewClusterClient()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get argocd clusters client: %s", err)
	}
	defer conn.Close()
	argoClusters, err := clApi.List(ctx, nil, nil)
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
	arlonClusterMap := make(map[string]*argoappapi.Application)
	for _, arlonClust := range arlonClusters.Items {
		arlonClusterMap[arlonClust.Name] = &arlonClust
	}

	// Get applications (applicationsets managed by Arlon)
	var appList appset.ApplicationSetList
	rqmt, err := labels.NewRequirement("arlon-type", selection.In, []string{"application"})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create requirement: %s", err)
	}
	sel := labels.NewSelector().Add(*rqmt)
	err = cli.List(ctx, &appList, &client.ListOptions{
		Namespace:     req.Namespace,
		LabelSelector: sel,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list applicationsets: %s", err)
	}
	validAppNames := sets.NewSet()
	for _, item := range appList.Items {
		validAppNames.Add(item.Name)
	}
	log.V(1).Info("app names counted", "count", len(appList.Items))

	// Get profiles
	var profList arlonv1.AppProfileList
	err = cli.List(ctx, &profList, nil)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list app profiles: %s", err)
	}
	log.V(1).Info("app profiles counted", "count", len(profList.Items))

	// Reconcile profiles
	profNames := sets.NewSet()
	for _, prof := range profList.Items {
		profNames.Add(prof.Name)
		dirty := false
		beforeInvalidNames := invalidAppNamesFromProfile(&prof)
		afterInvalidNames := sets.NewSet()
		for _, appName := range prof.Spec.AppNames {
			if !validAppNames.Contains(appName) {
				afterInvalidNames.Add(appName)
			}
		}
		if !beforeInvalidNames.Equal(afterInvalidNames) {
			prof.Status.InvalidAppNames = setToStringSlice(afterInvalidNames)
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
			err = cli.Status().Update(ctx, &prof)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update app profile: %s", err)
			}
			log.V(1).Info("updated app profile", "profileName", prof.Name)
		}
	}

	// Reconcile clusters
	for _, argoClust := range argoClusters.Items {
		if argoClust.Labels == nil {
			argoClust.Labels = make(map[string]string)
		}
		argoClustLabel := argoClust.Labels[arlonapp.ProfileLabelKey]
		dirty := false
		arlonClust := arlonClusterMap[argoClust.Name]
		if arlonClust == nil {
			// No corresponding arlon cluster. Could be an "external" cluster,
			// so allow the label to be managed independently.
			continue
		} else {
			// Arlon cluster exists. Ensure argocd cluster is labeled identically
			if arlonClust.Labels == nil {
				arlonClust.Labels = make(map[string]string)
			}
			arlonClustLabel := arlonClust.Labels[arlonapp.ProfileLabelKey]
			if arlonClustLabel != "" {
				// Arlon cluster has label
				if argoClustLabel != arlonClustLabel {
					log.V(1).Info("updating label on argo cluster to match arlon cluster's",
						"labelValue", arlonClustLabel)
					argoClust.Labels[arlonapp.ProfileLabelKey] = arlonClustLabel
					dirty = true
				}
			} else if argoClustLabel != "" {
				// Arlon cluster has no label but argo cluster has one
				log.V(1).Info("removing label from argo cluster because arlon cluster has none",
					"argoClusterName", argoClust.Name)
				delete(argoClust.Labels, arlonapp.ProfileLabelKey)
				dirty = true
			}
		}
		if dirty {
			_, err = clApi.Update(context.Background(), &clusterpkg.ClusterUpdateRequest{
				Cluster:       &argoClust,
				UpdatedFields: []string{"labels"},
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update argocd cluster: %s", err)
			}
		}
	}
	return ctrl.Result{}, nil
}

func invalidAppNamesFromProfile(prof *arlonv1.AppProfile) sets.Set {
	appNames := sets.NewSet()
	for _, appName := range prof.Status.InvalidAppNames {
		appNames.Add(appName)
	}
	return appNames
}

func setToStringSlice(s sets.Set) []string {
	var strs []string
	for elem := range s.Iter() {
		strs = append(strs, elem.(string))
	}
	return strs
}
