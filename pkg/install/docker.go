package install

type dockerInstaller struct {
}

func (d *dockerInstaller) EnsureRequisites() error {
	return nil
}

func (d *dockerInstaller) Bootstrap() error {
	return nil
}
