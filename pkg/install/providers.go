package install

import "errors"

type InstallerService interface {
	EnsureRequisites() error
	Bootstrap() error
}

func NewInstallerService(provider string) (InstallerService, error) {
	switch provider {
	case "aws":
		return &awsInstaller{}, nil
	default:
		return nil, errors.New("invalid provider")
	}
}
