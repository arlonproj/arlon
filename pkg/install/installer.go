package install

import "errors"

type InstallerService interface {
	EnsureRequisites() error
	Bootstrap() error
}

type ErrBootstrap struct {
	HardFail bool
	Message  string
}

func (e *ErrBootstrap) Error() string {
	return e.Message
}

func NewInstallerService(provider string) (InstallerService, error) {
	switch provider {
	case "aws":
		return &awsInstaller{}, nil
	case "docker":
		return &dockerInstaller{}, nil
	default:
		return nil, errors.New("invalid provider")
	}
}
