package appprofile

import (
	"context"
	"fmt"
	"github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/ctrlruntimeclient"
	restclient "k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/tabwriter"
)

func List(config *restclient.Config, ns string) (apslist []v1.AppProfile, err error) {
	cli, err := ctrlruntimeclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var apl v1.AppProfileList
	err = cli.List(context.Background(), &apl, &client.ListOptions{
		Namespace: ns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list application profiles: %s", err)
	}
	return apl.Items, nil
}

func ListToStdout(config *restclient.Config, ns string) error {
	profiles, err := List(config, ns)
	if err != nil {
		return fmt.Errorf("failed to list application profiles: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tAPPS\tHEALTH\tINVALID_APPS\n")
	for _, prof := range profiles {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			prof.Name,
			prof.Spec.AppNames,
			prof.Status.Health,
			prof.Status.InvalidAppNames,
		)
	}
	_ = w.Flush()
	return nil
}
