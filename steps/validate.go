package steps

import (
	"errors"

	"github.com/blang/semver"
	"github.com/wercker/wercker/util"
)

// isSemVer checks if the version adheres to the SemVer specification:
// http://semver.org/
func isSemVer(version string) bool {
	_, err := semver.Make(version)
	return err == nil
}

// ValidateManifest checks for some common issues, before sending the manifest
// to the Wercker steps server.
func ValidateManifest(manifest *StepManifest) error {
	var e []error

	if manifest.Name == "" {
		e = append(e, errors.New("Name cannot be empty"))
	}

	if manifest.Summary == "" {
		e = append(e, errors.New("Summary cannot be empty"))
	}

	if !isSemVer(manifest.Version) {
		e = append(e, errors.New("Version does not appear to be valid semver"))
	}

	return util.SqaushErrors(e)
}
