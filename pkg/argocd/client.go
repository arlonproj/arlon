package argocd

import (
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/util/errors"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
)

func NewArgocdClientOrDie() apiclient.Client {
	defaultLocalConfigPath, err := localconfig.DefaultLocalConfigPath()
	errors.CheckError(err)
	var argocdCliOpts apiclient.ClientOptions
	argocdCliOpts.ConfigPath = defaultLocalConfigPath
	return argocdclient.NewClientOrDie(&argocdCliOpts)
}
