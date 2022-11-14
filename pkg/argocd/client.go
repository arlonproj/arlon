package argocd

import (
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/util/errors"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
)

func NewArgocdClientOrDie(argocdConfigPath string) apiclient.Client {
	client, err := NewArgocdClient(argocdConfigPath)
	errors.CheckError(err)
	return client
}

func NewArgocdClient(argocdConfigPath string) (apiclient.Client, error) {
	if argocdConfigPath == "" {
		var err error
		argocdConfigPath, err = localconfig.DefaultLocalConfigPath()
		if err != nil {
			return nil, err
		}
	}
	var argocdCliOpts apiclient.ClientOptions
	argocdCliOpts.ConfigPath = argocdConfigPath
	return argocdclient.NewClient(&argocdCliOpts)
}
