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

package main

import (
	"flag"
	"os"

	"github.com/spf13/cobra"

	"github.com/arlonproj/arlon/cmd/bundle"
	"github.com/arlonproj/arlon/cmd/callhomecontroller"
	"github.com/arlonproj/arlon/cmd/cluster"
	"github.com/arlonproj/arlon/cmd/clusterspec"
	"github.com/arlonproj/arlon/cmd/controller"
	"github.com/arlonproj/arlon/cmd/list_clusters"
	"github.com/arlonproj/arlon/cmd/profile"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

func main() {
	command := &cobra.Command{
		Use:               "arlon",
		Short:             "Run the Arlon program",
		Long:              "Run the Arlon program",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			c.Println(c.UsageString())
		},
	}
	// don't display usage upon error
	command.SilenceUsage = true
	command.AddCommand(controller.NewCommand())
	command.AddCommand(callhomecontroller.NewCommand())
	command.AddCommand(list_clusters.NewCommand())
	command.AddCommand(bundle.NewCommand())
	command.AddCommand(profile.NewCommand())
	command.AddCommand(clusterspec.NewCommand())
	command.AddCommand(cluster.NewCommand())

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	// override default log level, which is initially set to 'debug'
	flag.Set("zap-log-level", "info")
	flag.Parse()
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
	args := flag.Args()
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
