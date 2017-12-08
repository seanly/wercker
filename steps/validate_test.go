package steps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isSemVer_Valid(t *testing.T) {
	versions := []string{
		"1.0.0",
		"1.0.0-beta1",
		"0.0.1-alpha.preview+123.github",
	}

	for _, version := range versions {
		actual := isSemVer(version)

		assert.True(t, actual, `isSemVer should return true for: "%s"`, version)
	}
}

func Test_isSemVer_Invalid(t *testing.T) {
	versions := []string{
		"",
		"1.0",
		"01.0.0",
		"v1.0.0",
		" 1.0.0 ",
	}

	for _, version := range versions {
		actual := isSemVer(version)

		assert.False(t, actual, `isSemVer should return false for: "%s"`, version)
	}
}
