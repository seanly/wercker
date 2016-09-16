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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/pborman/uuid"
	"github.com/wercker/wercker/util"
)

var (
	DEFAULT_BASE_URL = "https://app.wercker.com"
)

// GlobalOptions applicable to everything
type GlobalOptions struct {
	BaseURL    string
	Debug      bool
	Journal    bool
	Verbose    bool
	ShowColors bool

	// Auth
	AuthToken      string
	AuthTokenStore string
}

// guessAuthToken will attempt to read from the token store location if
// no auth token was provided
func guessAuthToken(c util.Settings, e *util.Environment, authTokenStore string) string {
	token, _ := c.GlobalString("auth-token")
	if token != "" {
		return token
	}
	if foundToken, _ := util.Exists(authTokenStore); !foundToken {
		return ""
	}

	tokenBytes, err := ioutil.ReadFile(authTokenStore)
	if err != nil {
		util.RootLogger().WithField("Logger", "Options").Errorln(err)
		return ""
	}
	return strings.TrimSpace(string(tokenBytes))
}

// NewGlobalOptions constructor
func NewGlobalOptions(c util.Settings, e *util.Environment) (*GlobalOptions, error) {
	baseURL, _ := c.GlobalString("base-url", DEFAULT_BASE_URL)
	baseURL = strings.TrimRight(baseURL, "/")
	debug, _ := c.GlobalBool("debug")
	journal, _ := c.GlobalBool("journal")
	verbose, _ := c.GlobalBool("verbose")
	// TODO(termie): switch negative flag
	showColors, _ := c.GlobalBool("no-colors")
	showColors = !showColors

	authTokenStore, _ := c.GlobalString("auth-token-store")
	authTokenStore = util.ExpandHomePath(authTokenStore, e.Get("HOME"))
	authToken := guessAuthToken(c, e, authTokenStore)

	// If debug is true, than force verbose and do not use colors.
	if debug {
		verbose = true
		showColors = false
	}

	return &GlobalOptions{
		BaseURL:    baseURL,
		Debug:      debug,
		Journal:    journal,
		Verbose:    verbose,
		ShowColors: showColors,

		AuthToken:      authToken,
		AuthTokenStore: authTokenStore,
	}, nil
}

// AWSOptions for our artifact storage
type AWSOptions struct {
	*GlobalOptions
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	S3Bucket           string
	S3PartSize         int64
}

// NewAWSOptions constructor
func NewAWSOptions(c util.Settings, e *util.Environment, globalOpts *GlobalOptions) (*AWSOptions, error) {
	awsAccessKeyID, _ := c.String("aws-access-key")
	awsRegion, _ := c.String("aws-region")
	awsSecretAccessKey, _ := c.String("aws-secret-key")
	s3Bucket, _ := c.String("s3-bucket")

	return &AWSOptions{
		GlobalOptions:      globalOpts,
		AWSAccessKeyID:     awsAccessKeyID,
		AWSRegion:          awsRegion,
		AWSSecretAccessKey: awsSecretAccessKey,
		S3Bucket:           s3Bucket,
		S3PartSize:         100 * 1024 * 1024, // 100 MB
	}, nil
}

// GitOptions for the users, mostly
type GitOptions struct {
	*GlobalOptions
	GitBranch     string
	GitCommit     string
	GitDomain     string
	GitOwner      string
	GitRepository string
}

func guessGitBranch(c util.Settings, e *util.Environment) string {
	branch, _ := c.String("git-branch")
	if branch != "" {
		return branch
	}

	projectPath := guessProjectPath(c, e)
	if projectPath == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	defer os.Chdir(cwd)
	os.Chdir(projectPath)

	git, err := exec.LookPath("git")
	if err != nil {
		return ""
	}

	var out bytes.Buffer
	cmd := exec.Command(git, "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return ""
	}
	return strings.Trim(out.String(), "\n")
}

func guessGitCommit(c util.Settings, e *util.Environment) string {
	commit, _ := c.String("git-commit")
	if commit != "" {
		return commit
	}

	projectPath := guessProjectPath(c, e)
	if projectPath == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	defer os.Chdir(cwd)
	os.Chdir(projectPath)

	git, err := exec.LookPath("git")
	if err != nil {
		return ""
	}

	var out bytes.Buffer
	cmd := exec.Command(git, "rev-parse", "HEAD")
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return ""
	}
	return strings.Trim(out.String(), "\n")
}

func guessGitOwner(c util.Settings, e *util.Environment) string {
	owner, _ := c.String("git-owner")
	if owner != "" {
		return owner
	}

	u, err := user.Current()
	if err == nil {
		owner = u.Username
	}
	return owner
}

func guessGitRepository(c util.Settings, e *util.Environment) string {
	repository, _ := c.String("git-repository")
	if repository != "" {
		return repository
	}
	// repository, err := guessApplicationName(c, env)
	// if err != nil {
	//   return ""
	// }
	return repository
}

// NewGitOptions constructor
func NewGitOptions(c util.Settings, e *util.Environment, globalOpts *GlobalOptions) (*GitOptions, error) {
	gitBranch := guessGitBranch(c, e)
	gitCommit := guessGitCommit(c, e)
	gitDomain, _ := c.String("git-domain")
	gitOwner := guessGitOwner(c, e)
	gitRepository := guessGitRepository(c, e)

	return &GitOptions{
		GlobalOptions: globalOpts,
		GitBranch:     gitBranch,
		GitCommit:     gitCommit,
		GitDomain:     gitDomain,
		GitOwner:      gitOwner,
		GitRepository: gitRepository,
	}, nil
}

// KeenOptions for our metrics
type KeenOptions struct {
	*GlobalOptions
	KeenProjectID       string
	KeenProjectWriteKey string
	ShouldKeenMetrics   bool
}

// NewKeenOptions constructor
func NewKeenOptions(c util.Settings, e *util.Environment, globalOpts *GlobalOptions) (*KeenOptions, error) {
	keenMetrics, _ := c.Bool("keen-metrics")
	keenProjectWriteKey, _ := c.String("keen-project-write-key")
	keenProjectID, _ := c.String("keen-project-id")

	if keenMetrics {
		if keenProjectWriteKey == "" {
			return nil, errors.New("keen-project-write-key is required")
		}

		if keenProjectID == "" {
			return nil, errors.New("keen-project-id is required")
		}
	}

	return &KeenOptions{
		GlobalOptions:       globalOpts,
		KeenProjectID:       keenProjectID,
		KeenProjectWriteKey: keenProjectWriteKey,
		ShouldKeenMetrics:   keenMetrics,
	}, nil
}

// ReporterOptions for our reporting
type ReporterOptions struct {
	*GlobalOptions
	ReporterHost string
	ReporterKey  string
	ShouldReport bool
}

// NewReporterOptions constructor
func NewReporterOptions(c util.Settings, e *util.Environment, globalOpts *GlobalOptions) (*ReporterOptions, error) {
	shouldReport, _ := c.Bool("report")
	reporterHost, _ := c.String("wercker-host")
	reporterKey, _ := c.String("wercker-token")

	if shouldReport {
		if reporterKey == "" {
			return nil, errors.New("wercker-token is required")
		}

		if reporterHost == "" {
			return nil, errors.New("wercker-host is required")
		}
	}

	return &ReporterOptions{
		GlobalOptions: globalOpts,
		ReporterHost:  reporterHost,
		ReporterKey:   reporterKey,
		ShouldReport:  shouldReport,
	}, nil
}

// PipelineOptions for builds and deploys
type PipelineOptions struct {
	*GlobalOptions
	*AWSOptions
	// *DockerOptions
	*GitOptions
	*KeenOptions
	*ReporterOptions

	// TODO(termie): i'd like to remove this, it is only used in a couple
	//               places by BasePipeline
	HostEnv *util.Environment

	BuildID      string
	DeployID     string
	PipelineID   string
	DeployTarget string
	Pipeline     string

	ApplicationID            string
	ApplicationName          string
	ApplicationOwnerName     string
	ApplicationStartedByName string

	ShouldCommit  bool
	Repository    string
	Tag           string
	Message       string
	ShouldStoreS3 bool

	WorkingDir string

	GuestRoot  string
	MntRoot    string
	ReportRoot string
	// will be set by pipeline when it initializes
	PipelineBasePath string

	ProjectID   string
	ProjectURL  string
	ProjectPath string

	CommandTimeout    int
	NoResponseTimeout int
	ShouldArtifacts   bool
	ShouldRemove      bool
	SourceDir         string
	IgnoreFile        string

	AttachOnError  bool
	DirectMount    bool
	EnableDevSteps bool
	PublishPorts   []string
	ExposePorts    bool
	EnableVolumes  bool
	WerckerYml     string
	Checkpoint     string

	DefaultsUsed PipelineDefaultsUsed
}

type PipelineDefaultsUsed struct {
	IgnoreFile bool
}

func guessApplicationID(c util.Settings, e *util.Environment, name string) string {
	id, _ := c.String("application-id")
	if id == "" {
		id = name
	}
	return id
}

// Some logic to guess the application name
func guessApplicationName(c util.Settings, e *util.Environment) (string, error) {
	applicationName, _ := c.String("application-name")
	if applicationName != "" {
		return applicationName, nil
	}

	// Otherwise, check our build target, it can be a url...
	target, _ := c.String("target")
	projectURL := ""
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		projectURL = target
		base := path.Base(projectURL)
		// Special handling for github tarballs
		if base == "tarball" {
			base = path.Base(path.Dir(projectURL))
		}
		ext := path.Ext(base)
		base = base[:len(ext)]
		return base, nil
	}

	// ... or a file path
	if target == "" {
		target = "."
	}
	stat, err := os.Stat(target)
	if err != nil || !stat.IsDir() {
		return "", fmt.Errorf("target '%s' is not a directory", target)
	}
	abspath, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return filepath.Base(abspath), nil
}

func guessApplicationOwnerName(c util.Settings, e *util.Environment) string {
	name, _ := c.String("application-owner-name")
	if name == "" {
		u, err := user.Current()
		if err == nil {
			name = u.Username
		}
	}
	if name == "" {
		name = "wercker"
	}
	return name
}

func guessMessage(c util.Settings, e *util.Environment) string {
	message, _ := c.String("message")
	return message
}

func guessTag(c util.Settings, e *util.Environment) string {
	tag, _ := c.String("tag")
	if tag == "" {
		tag = guessGitBranch(c, e)
	}
	tag = strings.Replace(tag, "/", "_", -1)
	return tag
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func guessProjectID(c util.Settings, e *util.Environment) string {
	projectID, _ := c.String("project-id")
	if projectID != "" {
		return projectID
	}

	// If this was going to fail it already failed and we exited
	name, _ := guessApplicationName(c, e)
	return name
}

func guessProjectPath(c util.Settings, e *util.Environment) string {
	target, _ := c.String("target")
	if looksLikeURL(target) {
		return ""
	}
	if target == "" {
		target = "."
	}
	abs, _ := filepath.Abs(target)
	return abs
}

func guessProjectURL(c util.Settings, e *util.Environment) string {
	target, _ := c.String("target")
	if !looksLikeURL(target) {
		return ""
	}
	return target
}

// NewPipelineOptions big-ass constructor
func NewPipelineOptions(c util.Settings, e *util.Environment) (*PipelineOptions, error) {
	globalOpts, err := NewGlobalOptions(c, e)
	if err != nil {
		return nil, err
	}

	// dockerOpts, err := NewDockerOptions(c, e, globalOpts)
	// if err != nil {
	//   return nil, err
	// }

	awsOpts, err := NewAWSOptions(c, e, globalOpts)
	if err != nil {
		return nil, err
	}

	gitOpts, err := NewGitOptions(c, e, globalOpts)
	if err != nil {
		return nil, err
	}

	keenOpts, err := NewKeenOptions(c, e, globalOpts)
	if err != nil {
		return nil, err
	}

	reporterOpts, err := NewReporterOptions(c, e, globalOpts)
	if err != nil {
		return nil, err
	}

	buildID, _ := c.String("build-id")
	deployID, _ := c.String("deploy-id")
	pipelineID := ""
	if deployID != "" {
		pipelineID = deployID
	} else {
		pipelineID = buildID
	}
	deployTarget, _ := c.String("deploy-target")
	pipeline, _ := c.String("pipeline")

	applicationName, err := guessApplicationName(c, e)
	if err != nil {
		return nil, err
	}
	applicationID := guessApplicationID(c, e, applicationName)
	applicationOwnerName := guessApplicationOwnerName(c, e)
	applicationStartedByName, _ := c.String("application-started-by-name")
	if applicationStartedByName == "" {
		applicationStartedByName = applicationOwnerName
	}

	repository, _ := c.String("commit")
	shouldCommit := (repository != "")
	tag := guessTag(c, e)
	message := guessMessage(c, e)
	shouldStoreS3, _ := c.Bool("store-s3")

	workingDir, _ := c.String("working-dir")
	workingDir, _ = filepath.Abs(workingDir)

	guestRoot, _ := c.String("guest-root")
	mntRoot, _ := c.String("mnt-root")
	reportRoot, _ := c.String("report-root")

	projectID := guessProjectID(c, e)
	projectPath := guessProjectPath(c, e)
	projectURL := guessProjectURL(c, e)

	if projectPath == workingDir {
		return nil, fmt.Errorf("Project path can't be the same as the working dir")
	}

	// These timeouts are given in minutes but we store them as milliseconds
	commandTimeoutFloat, _ := c.Float64("command-timeout")
	commandTimeout := int(commandTimeoutFloat * 1000 * 60)
	noResponseTimeoutFloat, _ := c.Float64("no-response-timeout")
	noResponseTimeout := int(noResponseTimeoutFloat * 1000 * 60)
	shouldArtifacts, _ := c.Bool("artifacts")
	// TODO(termie): switch negative flag
	shouldRemove, _ := c.Bool("no-remove")
	shouldRemove = !shouldRemove
	sourceDir, _ := c.String("source-dir")
	ignoreFile, ignoreFileSet := c.String("ignore-file")

	attachOnError, _ := c.Bool("attach-on-error")
	directMount, _ := c.Bool("direct-mount")
	enableDevSteps, _ := c.Bool("enable-dev-steps")
	// Deprecated
	publishPorts, _ := c.StringSlice("publish")
	exposePorts, _ := c.Bool("expose-ports")
	enableVolumes, _ := c.Bool("enable-volumes")
	werckerYml, _ := c.String("wercker-yml")
	checkpoint, _ := c.String("checkpoint")

	defaultsUsed := PipelineDefaultsUsed {
		IgnoreFile: !ignoreFileSet,
	}

	return &PipelineOptions{
		GlobalOptions: globalOpts,
		AWSOptions:    awsOpts,
		// DockerOptions:   dockerOpts,
		GitOptions:      gitOpts,
		KeenOptions:     keenOpts,
		ReporterOptions: reporterOpts,

		HostEnv: e,

		BuildID:      buildID,
		DeployID:     deployID,
		PipelineID:   pipelineID,
		DeployTarget: deployTarget,
		Pipeline:     pipeline,

		ApplicationID:            applicationID,
		ApplicationName:          applicationName,
		ApplicationOwnerName:     applicationOwnerName,
		ApplicationStartedByName: applicationStartedByName,

		Message:       message,
		Tag:           tag,
		Repository:    repository,
		ShouldCommit:  shouldCommit,
		ShouldStoreS3: shouldStoreS3,

		WorkingDir: workingDir,

		GuestRoot:  guestRoot,
		MntRoot:    mntRoot,
		ReportRoot: reportRoot,

		ProjectID:   projectID,
		ProjectURL:  projectURL,
		ProjectPath: projectPath,

		CommandTimeout:    commandTimeout,
		NoResponseTimeout: noResponseTimeout,
		ShouldArtifacts:   shouldArtifacts,
		ShouldRemove:      shouldRemove,
		SourceDir:         sourceDir,
		IgnoreFile:        ignoreFile,

		AttachOnError:  attachOnError,
		DirectMount:    directMount,
		EnableDevSteps: enableDevSteps,
		// Deprecated
		PublishPorts:  publishPorts,
		ExposePorts:   exposePorts,
		EnableVolumes: enableVolumes,
		WerckerYml:    werckerYml,
		Checkpoint:    checkpoint,

		DefaultsUsed: defaultsUsed,
	}, nil
}

// HostPath returns a path relative to the build root on the host.
func (o *PipelineOptions) HostPath(s ...string) string {
	return path.Join(o.BuildPath(), o.PipelineID, path.Join(s...))
}

// WorkingPath returns paths relative to our working dir (usually ".wercker")
func (o *PipelineOptions) WorkingPath(s ...string) string {
	return path.Join(o.WorkingDir, path.Join(s...))
}

// GuestPath returns a path relative to the build root on the guest.
func (o *PipelineOptions) GuestPath(s ...string) string {
	return path.Join(o.GuestRoot, path.Join(s...))
}

func (o *PipelineOptions) BasePath() string {
	basePath := o.GuestPath("source")
	if o.PipelineBasePath != "" {
		basePath = o.PipelineBasePath
	}
	return basePath
}

func (o *PipelineOptions) SourcePath() string {
	return path.Join(o.BasePath(), o.SourceDir)
}

// MntPath returns a path relative to the read-only mount root on the guest.
func (o *PipelineOptions) MntPath(s ...string) string {
	return path.Join(o.MntRoot, path.Join(s...))
}

// ReportPath returns a path relative to the report root on the guest.
func (o *PipelineOptions) ReportPath(s ...string) string {
	return path.Join(o.ReportRoot, path.Join(s...))
}

// ContainerPath returns the path where exported containers live
func (o *PipelineOptions) ContainerPath() string {
	return path.Join(o.WorkingDir, "containers")
}

// BuildPath returns the path where created builds live
func (o *PipelineOptions) BuildPath(s ...string) string {
	return path.Join(o.WorkingDir, "builds", path.Join(s...))
}

// CachePath returns the path for storing pipeline cache
func (o *PipelineOptions) CachePath() string {
	return path.Join(o.WorkingDir, "cache")
}

// ProjectDownloadPath returns the path where downloaded projects live
func (o *PipelineOptions) ProjectDownloadPath() string {
	return path.Join(o.WorkingDir, "projects")
}

// StepPath returns the path where downloaded steps live
func (o *PipelineOptions) StepPath() string {
	return path.Join(o.WorkingDir, "steps")
}

// IgnoreFilePath return the absolute path of the ignore file
func (o *PipelineOptions) IgnoreFilePath() string {
	expandedIgnoreFile := util.ExpandHomePath(o.IgnoreFile, o.HostEnv.Get("HOME"))

	if filepath.IsAbs(expandedIgnoreFile) {
	  return expandedIgnoreFile
	} else {
	  return path.Join(o.ProjectPath, o.IgnoreFile)
	}
}

// Options per Command

type optionsGetter func(*cli.Context, *util.Environment) (*PipelineOptions, error)

// NewBuildOptions constructor
func NewBuildOptions(c util.Settings, e *util.Environment) (*PipelineOptions, error) {
	pipelineOpts, err := NewPipelineOptions(c, e)
	if err != nil {
		return nil, err
	}
	if pipelineOpts.BuildID == "" {
		pipelineOpts.BuildID = uuid.NewRandom().String()
		pipelineOpts.PipelineID = pipelineOpts.BuildID
	}
	return pipelineOpts, nil
}

// NewDevOptions ctor
func NewDevOptions(c util.Settings, e *util.Environment) (*PipelineOptions, error) {
	pipelineOpts, err := NewBuildOptions(c, e)
	if err != nil {
		return nil, err
	}
	return pipelineOpts, nil
}

// NewCheckConfigOptions constructor
func NewCheckConfigOptions(c util.Settings, e *util.Environment) (*PipelineOptions, error) {
	pipelineOpts, err := NewPipelineOptions(c, e)
	if err != nil {
		return nil, err
	}
	return pipelineOpts, nil
}

// NewDeployOptions constructor
func NewDeployOptions(c util.Settings, e *util.Environment) (*PipelineOptions, error) {
	pipelineOpts, err := NewPipelineOptions(c, e)
	if err != nil {
		return nil, err
	}
	// default to last build output if none defined
	target, _ := c.String("target")
	if target == "" {
		found, err := util.Exists("./.wercker/latest/output")
		if err == nil && found {
			util.RootLogger().Println("No target specified, using recent build output.")
			pipelineOpts.ProjectPath, _ = filepath.Abs("./.wercker/latest/output")
		}
	}

	// if the deploy target path does not have a wercker.yml, use the current one
	werckerYml, _ := c.String("wercker-yml")
	if werckerYml == "" {
		found, _ := util.Exists(filepath.Join(pipelineOpts.ProjectPath, "wercker.yml"))
		if !found {
			pipelineOpts.WerckerYml = "./wercker.yml"
		}
	}

	if pipelineOpts.DeployID == "" {
		pipelineOpts.DeployID = uuid.NewRandom().String()
		pipelineOpts.PipelineID = pipelineOpts.DeployID
	}
	return pipelineOpts, nil
}

// DetectOptions for detect command
type DetectOptions struct {
	*GlobalOptions
}

// NewDetectOptions constructor
func NewDetectOptions(c util.Settings, e *util.Environment) (*DetectOptions, error) {
	globalOpts, err := NewGlobalOptions(c, e)
	if err != nil {
		return nil, err
	}
	return &DetectOptions{globalOpts}, nil
}

// InspectOptions for inspect command
type InspectOptions struct {
	*PipelineOptions
}

// NewInspectOptions constructor
func NewInspectOptions(c util.Settings, e *util.Environment) (*InspectOptions, error) {
	pipelineOpts, err := NewPipelineOptions(c, e)
	if err != nil {
		return nil, err
	}
	return &InspectOptions{pipelineOpts}, nil
}

// LoginOptions for the login command
type LoginOptions struct {
	*GlobalOptions
}

// NewLoginOptions constructor
func NewLoginOptions(c util.Settings, e *util.Environment) (*LoginOptions, error) {
	globalOpts, err := NewGlobalOptions(c, e)
	if err != nil {
		return nil, err
	}
	return &LoginOptions{globalOpts}, nil
}

// LogoutOptions for the login command
type LogoutOptions struct {
	*GlobalOptions
}

// NewLogoutOptions constructor
func NewLogoutOptions(c util.Settings, e *util.Environment) (*LogoutOptions, error) {
	globalOpts, err := NewGlobalOptions(c, e)
	if err != nil {
		return nil, err
	}
	return &LogoutOptions{globalOpts}, nil
}

// PullOptions for the pull command
type PullOptions struct {
	*GlobalOptions
	// *DockerOptions

	Repository string
	Branch     string
	Commit     string
	Status     string
	Result     string
	Output     string
	Load       bool
	Force      bool
}

// NewPullOptions constructor
func NewPullOptions(c util.Settings, e *util.Environment) (*PullOptions, error) {
	globalOpts, err := NewGlobalOptions(c, e)
	if err != nil {
		return nil, err
	}

	// dockerOpts, err := NewDockerOptions(c, e, globalOpts)
	// if err != nil {
	//   return nil, err
	// }

	repository, _ := c.String("target")
	output, _ := c.String("output")
	outputDir, err := filepath.Abs(output)
	if err != nil {
		return nil, err
	}
	branch, _ := c.String("branch")
	status, _ := c.String("status")
	result, _ := c.String("result")
	load, _ := c.Bool("load")
	force, _ := c.Bool("force")

	return &PullOptions{
		GlobalOptions: globalOpts,
		// DockerOptions: dockerOpts,

		Repository: repository,
		Branch:     branch,
		Status:     status,
		Result:     result,
		Output:     outputDir,
		Load:       load,
		Force:      force,
	}, nil
}

// VersionOptions contains the options associated with the version
// command.
type VersionOptions struct {
	OutputJSON     bool
	BetaChannel    bool
	CheckForUpdate bool
}

// NewVersionOptions constructor
func NewVersionOptions(c util.Settings, e *util.Environment) (*VersionOptions, error) {
	json, _ := c.Bool("json")
	beta, _ := c.Bool("beta")
	noUpdateCheck, _ := c.Bool("no-update-check")

	return &VersionOptions{
		OutputJSON:     json,
		BetaChannel:    beta,
		CheckForUpdate: !noUpdateCheck,
	}, nil
}
