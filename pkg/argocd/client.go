package argocd

import (
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/util/errors"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
)

func NewArgocdClientOrDie(argocdConfigPath string) apiclient.Client {
	if argocdConfigPath == "" {
		var err error
		argocdConfigPath, err = localconfig.DefaultLocalConfigPath()
		errors.CheckError(err)
	}
	var argocdCliOpts apiclient.ClientOptions
	argocdCliOpts.ConfigPath = argocdConfigPath
	return argocdclient.NewClientOrDie(&argocdCliOpts)
}
