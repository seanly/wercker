package steps

type StepManifest struct {
	// name of the step
	Name string `json:"name,omitempty"`
	// version of the step
	Version string `json:"version,omitempty"`
	// summary of the step
	Summary string `json:"summary,omitempty"`
	// tags of the step
	Tags []string `json:"tags,omitempty"`
	// properties of the step
	Properties []*StepProperty `json:"properties,omitempty"`
}

type StepProperty struct {
	// name of the property
	Name string `json:"name,omitempty"`
	// type of the property
	Type string `json:"type,omitempty"`
	// required whether the property is required or not
	Required bool `json:"required,omitempty"`
	// default property
	Default string `json:"default,omitempty"`
}
