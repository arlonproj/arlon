package ngprofile

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	restclient "k8s.io/client-go/rest"
	"os"
	"text/tabwriter"
)

type NgProfile struct {
	Name string
	Apps []string
}

func List(config *restclient.Config, ns string) (ngplist []NgProfile, err error) {
	apps, err := app.List(config, ns)
	if err != nil {
		err = fmt.Errorf("failed to list apps: %s", err)
		return
	}
	profMap := make(map[string]*NgProfile)
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
	for _, prof := range profMap {
		ngplist = append(ngplist, *prof)
	}
	return
}

func ListToStdout(config *restclient.Config, ns string) error {
	profiles, err := List(config, ns)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tAPPS\n")
	for _, prof := range profiles {
		_, _ = fmt.Fprintf(w, "%s\t%s\n",
			prof.Name,
			prof.Apps,
		)
	}
	_ = w.Flush()
	return nil
}
