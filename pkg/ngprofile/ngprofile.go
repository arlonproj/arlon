package ngprofile

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/arlonproj/arlon/pkg/app"
	restclient "k8s.io/client-go/rest"
	"os"
	"text/tabwriter"
)

type NgProfile struct {
	Name     string
	Apps     []string
	Clusters []string
}

// -----------------------------------------------------------------------------

func List(
	config *restclient.Config,
	ns string,
	argoIf argoclient.Client,
) (ngplist []NgProfile, err error) {

	profMap, err := Enumerate(config, ns)
	if err != nil {
		err = fmt.Errorf("failed to enumerate profiles: %s", err)
		return
	}
	err = augmentWithClusterInfo(argoIf, profMap)
	if err != nil {
		err = fmt.Errorf("failed to augment profiles with cluster infor: %s", err)
		return
	}
	for _, prof := range profMap {
		ngplist = append(ngplist, *prof)
	}
	return
}

// -----------------------------------------------------------------------------

func Enumerate(config *restclient.Config, ns string) (profMap map[string]*NgProfile, err error) {
	apps, err := app.List(config, ns)
	if err != nil {
		err = fmt.Errorf("failed to list apps: %s", err)
		return
	}
	profMap = make(map[string]*NgProfile)
	for _, appl := range apps {
		if len(appl.Spec.Generators) != 1 || appl.Spec.Generators[0].Clusters == nil {
			continue
		}
		me := appl.Spec.Generators[0].Clusters.Selector.MatchExpressions
		if len(me) != 1 {
			continue
		}
		if me[0].Key != app.ProfileLabelKey {
			continue
		}
		for _, profName := range me[0].Values {
			if profMap[profName] == nil {
				profMap[profName] = &NgProfile{
					Name: profName,
					Apps: []string{appl.Name},
				}
			} else {
				profMap[profName].Apps = append(profMap[profName].Apps, appl.Name)
			}
		}
	}
	return
}

// -----------------------------------------------------------------------------

func augmentWithClusterInfo(
	argoIf argoclient.Client,
	profMap map[string]*NgProfile,
) error {
	conn, clustIf, err := argoIf.NewClusterClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd cluster client: %s", err)
	}
	defer conn.Close()
	clist, err := clustIf.List(context.Background(), &clusterpkg.ClusterQuery{})
	if err != nil {
		return fmt.Errorf("failed to list argocd clusters: %s", err)
	}
	for _, clust := range clist.Items {
		profName := clust.Labels[app.ProfileLabelKey]
		if profName == "" {
			continue
		}
		prof := profMap[profName]
		if prof == nil {
			continue
		}
		prof.Clusters = append(prof.Clusters, clust.Name)
	}
	return nil
}

// -----------------------------------------------------------------------------

func ListToStdout(
	config *restclient.Config,
	ns string,
	argoIf argoclient.Client,
) error {
	profiles, err := List(config, ns, argoIf)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tAPPS\tCLUSTERS\n")
	for _, prof := range profiles {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			prof.Name,
			prof.Apps,
			prof.Clusters,
		)
	}
	_ = w.Flush()
	return nil
}

// -----------------------------------------------------------------------------
