package install

type dockerInstaller struct {
}

// haven't encountered any soft failures when installing capd(yet)
func (d *dockerInstaller) recoverOnFail(_ string) bool {
	return false
}

func (d *dockerInstaller) EnsureRequisites() error {
	return nil
}

func (d *dockerInstaller) Bootstrap() error {
	return nil
}
