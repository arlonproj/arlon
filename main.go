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
	"arlon.io/arlon/cmd/bundle"
	"arlon.io/arlon/cmd/controller"
	"arlon.io/arlon/cmd/list_clusters"
	"arlon.io/arlon/cmd/profile"
	"fmt"
	"github.com/spf13/cobra"
	"os"

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
		},
	}

	command.AddCommand(controller.NewCommand())
	command.AddCommand(list_clusters.NewCommand())
	command.AddCommand(bundle.NewCommand())
	command.AddCommand(profile.NewCommand())

	/*
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	*/

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
	if err := command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
