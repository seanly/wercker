//   Copyright 2016 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package core

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/wercker/wercker/util"
)

// RawBoxConfig is the unwrapper for BoxConfig
type RawBoxConfig struct {
	*BoxConfig
}

// BoxConfig is the type for boxes in the config
type BoxConfig struct {
	ID         string
	Name       string
	Tag        string
	Cmd        string
	Env        map[string]string
	Ports      []string
	Username   string
	Password   string
	Registry   string
	Entrypoint string
	URL        string
	Volumes    string
}

// IsExternal tells us if the box (service) is located on disk
func (c *BoxConfig) IsExternal() bool {
	return c.URL != "" && strings.HasPrefix(c.URL, "file://")
}

// UnmarshalYAML first attempts to unmarshal as a string to ID otherwise
// attempts to unmarshal to the whole struct
func (r *RawBoxConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.BoxConfig = &BoxConfig{}
	err := unmarshal(&r.BoxConfig.ID)
	if err != nil {
		err = unmarshal(&r.BoxConfig)
	}
	return err
}

// RawStepConfig is our unwrapper for config steps
type RawStepConfig struct {
	*StepConfig
}

// StepConfig holds our step configs
type StepConfig struct {
	ID         string
	Cwd        string
	Name       string
	Data       map[string]string
	Checkpoint string
}

// ifaceToString takes a value from yaml and makes it a string (currently
// supported: string, int, bool). Returns an empty string if the type is not
// supported.
func ifaceToString(dataValue interface{}) string {
	switch v := dataValue.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int32:
		i := int64(v)
		return strconv.FormatInt(i, 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ("")
	}
}

// UnmarshalYAML is fun, for this one as we're supporting three different
// types of yaml structures, a string, a map[string]map[string]string,
// and a map[string]string, these basically equate to these three styles
// of specifying the step that people commonly use:
//   steps:
//    - string-step  # this parses as a string
//    - script:      # this parses as a map[string]map[string]string
//        code: done right
//    - script:      # this parses as a map[string]string
//      code: done wrong
func (r *RawStepConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.StepConfig = &StepConfig{}

	// First up, check whether we're just a string
	err := unmarshal(&r.StepConfig.ID)
	if err == nil {
		return nil
	}

	// Next check whether we are a one-key map
	var stepID string
	stepData := make(map[string]string)
	var topMap yaml.MapSlice
	err = unmarshal(&topMap)
	if len(topMap) == 1 {
		// The only item's key will be the stepID, value is data
		item := topMap[0]
		stepID = item.Key
		interData, ok := item.Value.(yaml.MapSlice)
		if !ok {
			return fmt.Errorf("Step %s is empty", item.Key)
		}
		for _, item := range interData {
			stepData[item.Key] = ifaceToString(item.Value)
		}
	} else {
		// Otherwise the first element's key is the id, and the rest
		// of the elements are the data
		// TODO(termie): Throw a deprecation/bad usage warning
		firstItem := topMap[0]
		stepID = firstItem.Key
		for _, item := range topMap[1:] {
			stepData[item.Key] = ifaceToString(item.Value)
		}
	}

	r.ID = stepID
	// At this point we should know the ID and have a map[string]string
	// to work with to get the rest of the data
	if v, ok := stepData["cwd"]; ok {
		r.Cwd = v
		delete(stepData, "cwd")
	}
	if v, ok := stepData["name"]; ok {
		r.Name = v
		delete(stepData, "name")
	}
	if v, ok := stepData["checkpoint"]; ok {
		r.Checkpoint = v
		delete(stepData, "checkpoint")
	}
	r.Data = stepData
	return nil
}

// RawStepsConfig is a list of RawStepConfigs
type RawStepsConfig []*RawStepConfig

// RawPipelineConfig is our unwrapper for PipelineConfig
type RawPipelineConfig struct {
	*PipelineConfig
}

// PipelineConfig is for any pipeline sections
// StepsMap is for compat with the multiple deploy target configs
// TODO(termie): it would be great to deprecate this behavior and switch
//               to multiple pipelines instead
type PipelineConfig struct {
	Box        *RawBoxConfig
	Steps      RawStepsConfig
	AfterSteps RawStepsConfig `yaml:"after-steps"`
	StepsMap   map[string][]*RawStepConfig
	Services   []*RawBoxConfig `yaml:"services"`
	BasePath   string          `yaml:"base-path"`
}

var pipelineReservedWords = map[string]struct{}{
	"box":         struct{}{},
	"services":    struct{}{},
	"steps":       struct{}{},
	"after-steps": struct{}{},
	"base-path":   struct{}{},
}

// UnmarshalYAML in this case is a little involved due to the myriad shapes our
// data can take for deploys (unfortunately), so we have to pretend the data is
// a map for a while and do a marshal/unmarshal hack to parse the subsections
func (r *RawPipelineConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First get the fields we know and love
	r.PipelineConfig = &PipelineConfig{
		StepsMap: make(map[string][]*RawStepConfig),
	}
	err := unmarshal(r.PipelineConfig)
	if err != nil {
		return err
	}

	// Having a slash in the path will cause sources to end up in /pipeline/source/source.
	// Remove a potential trailing slash
	r.PipelineConfig.BasePath = strings.TrimSuffix(r.PipelineConfig.BasePath, "/")

	// Then treat it like a map to get the extra fields
	m := map[string]interface{}{}
	err = unmarshal(&m)
	if err != nil {
		return err
	}
	for k, v := range m {
		// Skip the fields we already know
		if _, ok := pipelineReservedWords[k]; ok {
			continue
		}

		// Marshal the data so we can use the unmarshal logic on it
		b, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		// Finally, unmarshal each section as steps and add it to our map
		var otherSteps []*RawStepConfig
		err = yaml.Unmarshal(b, &otherSteps)
		if err != nil {
			return &yaml.TypeError{Errors: []string{fmt.Sprintf("Invalid extra key in pipeline, %s is not a list of steps", k)}}
		}
		r.PipelineConfig.StepsMap[k] = otherSteps
	}
	return nil
}

// Config is the data type for wercker.yml
type Config struct {
	Box               *RawBoxConfig   `yaml:"box"`
	CommandTimeout    int             `yaml:"command-timeout"`
	NoResponseTimeout int             `yaml:"no-response-timeout"`
	Services          []*RawBoxConfig `yaml:"services"`
	SourceDir         string          `yaml:"source-dir"`
	IgnoreFile        string          `yaml:"ignore-file"`
	PipelinesMap      map[string]*RawPipelineConfig
}

// RawConfig is the unwrapper for Config
type RawConfig struct {
	*Config
}

var configReservedWords = map[string]struct{}{
	"box":                 struct{}{},
	"command-timeout":     struct{}{},
	"no-response-timeout": struct{}{},
	"services":            struct{}{},
	"source-dir":          struct{}{},
}

// UnmarshalYAML in this case is a little involved due to the myriad shapes our
// data can take for deploys (unfortunately), so we have to pretend the data is
// a map for a while and do a marshal/unmarshal hack to parse the subsections
func (r *RawConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First get the fields we know and love
	r.Config = &Config{
		PipelinesMap: make(map[string]*RawPipelineConfig),
	}
	err := unmarshal(r.Config)

	// Then treat it like a map to get the extra fields
	m := map[string]*RawPipelineConfig{}
	err = unmarshal(&m)
	if err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
	}

	for k, v := range m {
		// Skip the fields we already know
		if _, ok := configReservedWords[k]; ok {
			continue
		}

		r.Config.PipelinesMap[k] = v
	}
	return nil
}

// IsValid ensures that the underlying Config is populated properly
func (r *RawConfig) IsValid() error {
	if r.Config == nil {
		return fmt.Errorf("Your wercker.yml is empty.")
	}

	return nil
}

func findYaml(searchDirs []string) (string, error) {
	possibleYaml := []string{"ewok.yml", "wercker.yml", ".wercker.yml"}

	for _, v := range searchDirs {
		for _, y := range possibleYaml {
			possibleYaml := path.Join(v, y)
			ymlExists, err := util.Exists(possibleYaml)
			if err != nil {
				return "", err
			}
			if !ymlExists {
				continue
			}
			return possibleYaml, nil
		}
	}
	return "", fmt.Errorf("No wercker.yml found")
}

// ReadWerckerYaml will try to find a wercker.yml file and return its bytes.
// TODO(termie): If allowDefault is true it will try to generate a
// default yaml file by inspecting the project.
func ReadWerckerYaml(searchDirs []string, allowDefault bool) ([]byte, error) {
	foundYaml, err := findYaml(searchDirs)
	if err != nil {
		return nil, err
	}

	// TODO(termie): If allowDefault, we'd generate something here
	// if !allowDefault && !found {
	//   return nil, errors.New("No wercker.yml found and no defaults allowed.")
	// }

	return ioutil.ReadFile(foundYaml)
}

// ConfigFromYaml reads a []byte as yaml and turn it into a Config object
func ConfigFromYaml(file []byte) (*Config, error) {
	var m RawConfig
	err := yaml.Unmarshal(file, &m)
	// also need to ensure the RawConfig is valid before sending it back
	if err == nil {
		err = m.IsValid()
	}
	if err != nil {
		errStr := err.Error()
		err = fmt.Errorf("Error parsing your wercker.yml:\n  %s", errStr)
		return nil, err
	}

	return m.Config, nil
}
