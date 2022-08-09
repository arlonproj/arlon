package bundle

import (
	"errors"
	"regexp"
)

const (
	maxLenRFC1123 = 63
)

var (
	// taken from https://github.com/kubernetes/kubernetes/issues/94088
	regexRFC1123   = regexp.MustCompile(`(?m)^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	ErrInvalidName = errors.New("invalid resource name")
)

func IsValidK8sName(name string) bool {
	if len(name) == 0 || len(name) > maxLenRFC1123 {
		return false
	}
	return regexRFC1123.MatchString(name)
}
