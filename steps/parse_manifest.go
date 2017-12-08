package steps

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// ParseManifest parse b as a StepManifest.
func ParseManifest(b []byte) (*StepManifest, error) {
	var manifest StepManifest
	err := yaml.Unmarshal(b, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ParseManifestReader first reads all of r into memory before using
// ParseManifest to unmarshall the content.
func ParseManifestReader(r io.Reader) (*StepManifest, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseManifest(b)
}
