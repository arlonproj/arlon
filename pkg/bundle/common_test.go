package bundle

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsValidK8sName(t *testing.T) {
	testCases := []struct {
		Name     string
		Desc     string
		Expected bool
	}{
		{
			Name:     "this-is-a-valid-k8s-name",
			Desc:     "ValidName",
			Expected: true,
		},
		{
			Name:     "this-is-an-invalid-k8s-name+",
			Desc:     "ContainsExtraSymbol",
			Expected: false,
		},
		{
			Name:     "THIS-Has-uppercase",
			Desc:     "UppercaseName",
			Expected: false,
		},
		{
			Name:     "thisisaveryveryverylongk8swannabenamewhichistotallyinvalidandnotallowedthiswillreturnfalseandiamsureofit",
			Desc:     "LongName",
			Expected: false,
		},
		{
			Name:     "gdrjnk+(gd",
			Desc:     "InvalidChars",
			Expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			res := IsValidK8sName(tc.Name)
			require.Equal(t, tc.Expected, res)
		})
	}
}
