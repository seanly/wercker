package steps

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseManifest_Valid(t *testing.T) {
	tests := []string{
		"valid.yml",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", "step_manifests", test))
			require.NoError(t, err, "Unable to open step manifest")

			manifest, err := ParseManifestReader(f)
			assert.NoError(t, err)
			assert.NotNil(t, manifest)
		})
	}
}

func Test_ParseManifest_Invalid(t *testing.T) {
	tests := []string{
		"invalid.yml",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", "step_manifests", test))
			require.NoError(t, err, "Unable to open step manifest")

			manifest, err := ParseManifestReader(f)
			assert.Error(t, err)
			assert.Nil(t, manifest)
		})
	}
}
