package install

import "errors"

type InstallerService interface {
	EnsureRequisites() error
	Bootstrap() error
	recoverOnFail(message string) bool
}

type ErrBootstrap struct {
	HardFail bool
	Message  string
}

func (e *ErrBootstrap) Error() string {
	return e.Message
}

func NewInstallerService(provider string, silence bool) (InstallerService, error) {
	switch provider {
	case "aws":
		return &awsInstaller{silence: silence}, nil
	case "docker":
		return &dockerInstaller{}, nil
	default:
		return nil, errors.New("invalid provider")
	}
}
