package bundle

type Bundle struct {
	Name string
	Data []byte
	// The following are only set on reference type bundles
	RepoUrl string
	RepoPath string
}


