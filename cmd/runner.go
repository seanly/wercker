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

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/monochromegane/go-gitignore"
	"github.com/pborman/uuid"
	"github.com/termie/go-shutil"
	"github.com/wercker/wercker/core"
	"github.com/wercker/wercker/docker"
	"github.com/wercker/wercker/event"
	"github.com/wercker/wercker/util"
	"golang.org/x/net/context"
)

// pipelineGetter is a function that will fetch the appropriate pipeline
// object from the Config.
type pipelineGetter func(*core.Config, *core.PipelineOptions, *dockerlocal.DockerOptions) (core.Pipeline, error)

// GetDevPipelineFactory makes dev pipelines out of arbitrarily
// named config sections
func GetDevPipelineFactory(name string) func(*core.Config, *core.PipelineOptions, *dockerlocal.DockerOptions) (core.Pipeline, error) {
	return func(config *core.Config, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (core.Pipeline, error) {
		builder := NewDockerBuilder(options, dockerOptions)
		_, ok := config.PipelinesMap[name]
		if !ok {
			return nil, fmt.Errorf("No pipeline named %s", name)
		}
		return dockerlocal.NewDockerBuild(name, config, options, dockerOptions, builder)
	}
}

// GetBuildPipelineFactory makes build pipelines out of arbitrarily
// named config sections
func GetBuildPipelineFactory(name string) func(*core.Config, *core.PipelineOptions, *dockerlocal.DockerOptions) (core.Pipeline, error) {
	return func(config *core.Config, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (core.Pipeline, error) {
		builder := NewDockerBuilder(options, dockerOptions)
		_, ok := config.PipelinesMap[name]
		if !ok {
			return nil, fmt.Errorf("No pipeline named %s", name)
		}
		return dockerlocal.NewDockerBuild(name, config, options, dockerOptions, builder)
	}
}

// GetDeployPipelineFactory makes deploy pipelines out of arbitrarily
// named config sections
func GetDeployPipelineFactory(name string) func(*core.Config, *core.PipelineOptions, *dockerlocal.DockerOptions) (core.Pipeline, error) {
	return func(config *core.Config, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (core.Pipeline, error) {
		builder := NewDockerBuilder(options, dockerOptions)
		_, ok := config.PipelinesMap[name]
		if !ok {
			return nil, fmt.Errorf("No pipeline named %s", name)
		}
		return dockerlocal.NewDockerDeploy(name, config, options, dockerOptions, builder)
	}
}

// Runner is the base type for running the pipelines.
type Runner struct {
	options       *core.PipelineOptions
	dockerOptions *dockerlocal.DockerOptions
	literalLogger *event.LiteralLogHandler
	metrics       *event.MetricsEventHandler
	reporter      *event.ReportHandler
	getPipeline   pipelineGetter
	logger        *util.LogEntry
	emitter       *core.NormalizedEmitter
	formatter     *util.Formatter
}

// NewRunner from global options
func NewRunner(ctx context.Context, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions, getPipeline pipelineGetter) (*Runner, error) {
	e, err := core.EmitterFromContext(ctx)
	if err != nil {
		return nil, err
	}
	logger := util.RootLogger().WithField("Logger", "Runner")
	// h, err := NewLogHandler()
	// if err != nil {
	//   p.logger.WithField("Error", err).Panic("Unable to LogHandler")
	// }
	// h.ListenTo(e)

	if options.Debug {
		dh := core.NewDebugHandler()
		dh.ListenTo(e)
	}

	l, err := event.NewLiteralLogHandler(options)
	if err != nil {
		logger.WithField("Error", err).Panic("Unable to event.LiteralLogHandler")
	}
	l.ListenTo(e)

	var mh *event.MetricsEventHandler
	if options.ShouldKeenMetrics {
		mh, err = event.NewMetricsHandler(options)
		if err != nil {
			logger.WithField("Error", err).Panic("Unable to MetricsHandler")
		}
		mh.ListenTo(e)
	}

	var r *event.ReportHandler
	if options.ShouldReport {
		r, err := event.NewReportHandler(options.ReporterHost, options.ReporterKey)
		if err != nil {
			logger.WithField("Error", err).Panic("Unable to event.ReportHandler")
		}
		r.ListenTo(e)
	}

	return &Runner{
		options:       options,
		dockerOptions: dockerOptions,
		literalLogger: l,
		metrics:       mh,
		reporter:      r,
		getPipeline:   getPipeline,
		logger:        logger,
		emitter:       e,
		formatter:     &util.Formatter{options.GlobalOptions.ShowColors},
	}, nil
}

// ProjectDir returns the directory where we expect to find the code for this project
func (p *Runner) ProjectDir() string {
	if p.options.DirectMount {
		return p.options.ProjectPath
	}
	return fmt.Sprintf("%s/%s", p.options.ProjectDownloadPath(), p.options.ApplicationID)
}

// EnsureCode makes sure the code is in the ProjectDir.
// NOTE(termie): When launched by kiddie-pool the ProjectPath will be
// set to the location where grappler checked out the code and the copy
// will be a little superfluous, but in the case where this is being
// run in Single Player Mode this copy is necessary to avoid screwing
// with the local dir.
func (p *Runner) EnsureCode() (string, error) {
	projectDir := p.ProjectDir()
	if p.options.DirectMount {
		return projectDir, nil
	}

	// If the target is a tarball feetch and build that
	if p.options.ProjectURL != "" {
		resp, err := util.FetchTarball(p.options.ProjectURL)
		if err != nil {
			return projectDir, err
		}
		err = util.Untargzip(projectDir, resp.Body)
		if err != nil {
			return projectDir, err
		}
	} else {
		// We were pointed at a path with ProjectPath, copy it to projectDir

		ignoreFiles := []string{
			p.options.WorkingDir,
		}

		oldbuilds, _ := filepath.Abs("./_builds")
		oldprojects, _ := filepath.Abs("./_projects")
		oldsteps, _ := filepath.Abs("./_steps")
		oldcache, _ := filepath.Abs("./_cache")
		oldcontainers, _ := filepath.Abs("./_containers")
		deprecatedPaths := []string{
			oldbuilds,
			oldprojects,
			oldsteps,
			oldcache,
			oldcontainers,
		}

		var err error

		var ignoreFile, _ = gitignore.NewGitIgnore(p.options.IgnoreFilePath())

		// Make sure we don't accidentally recurse or copy extra files
		ignoreFunc := func(src string, files []os.FileInfo) []string {
			ignores := []string{}
			for _, file := range files {
				abspath, err := filepath.Abs(filepath.Join(src, file.Name()))
				if err != nil {
					// Something went sufficiently wrong
					panic(err)
				}
				if util.ContainsString(ignoreFiles, abspath) || (ignoreFile != nil && ignoreFile.Match(abspath, file.IsDir())) {
					ignores = append(ignores, file.Name())
				}

				// TODO(termie): remove this warning after a while
				if util.ContainsString(deprecatedPaths, abspath) {
					p.logger.Warnln(fmt.Sprintf("Not ignoring deprecated runtime path, %s. You probably want to delete it so it doesn't get copied into your container. Runtime files are now stored under '.wercker' by default. This message will go away in a future update.", file.Name()))
				}
			}
			return ignores
		}
		copyOpts := &shutil.CopyTreeOptions{Ignore: ignoreFunc, CopyFunction: shutil.Copy, Symlinks: true}
		os.Rename(projectDir, fmt.Sprintf("%s-%s", projectDir, uuid.NewRandom().String()))
		err = shutil.CopyTree(p.options.ProjectPath, projectDir, copyOpts)
		if err != nil {
			return projectDir, err
		}
	}
	return projectDir, nil
}

// CleanupOldBuilds removes old builds and keeps the latest 2
func (p *Runner) CleanupOldBuilds() error {
	// how many recent builds to keep
	const keepDirs = 2

	buildPath := p.options.BuildPath()

	builds, err := ioutil.ReadDir(buildPath)
	if err != nil {
		return err
	}

	// remove files (.DS_Store etc)
	for i, f := range builds {
		if !f.IsDir() {
			builds = append(builds[:i], builds[i+1:]...)
		}
	}

	util.SortByModDate(builds)

	if len(builds) < keepDirs {
		// nothing to do
		return nil
	}

	cleanup := builds[keepDirs:]

	// only clean up builds older than 24h
	oldFile := time.Now().Add(time.Hour * -24)
	for _, f := range cleanup {
		if f.ModTime().Before(oldFile) {
			os.RemoveAll(path.Join(buildPath, f.Name()))
		}
	}

	return err
}

// GetConfig parses and returns the wercker.yml file.
func (p *Runner) GetConfig() (*core.Config, string, error) {
	// Return a []byte of the yaml we find or create.
	var werckerYaml []byte
	var err error
	if p.options.WerckerYml != "" {
		werckerYaml, err = ioutil.ReadFile(p.options.WerckerYml)
		if err != nil {
			return nil, "", err
		}
	} else {
		werckerYaml, err = core.ReadWerckerYaml([]string{p.ProjectDir()}, false)
		if err != nil {
			return nil, "", err
		}
	}

	// Parse that bad boy.
	rawConfig, err := core.ConfigFromYaml(werckerYaml)
	if err != nil {
		return nil, "", err
	}

	// Add some options to the global config
	if rawConfig.SourceDir != "" {
		p.options.SourceDir = rawConfig.SourceDir
	}

	// Only use the ignore file from the config when it is not empty and not defined as a command-line option
	if rawConfig.IgnoreFile != "" && p.options.DefaultsUsed.IgnoreFile {
	  p.options.IgnoreFile = rawConfig.IgnoreFile
	}

	MaxCommandTimeout := 60    // minutes
	MaxNoResponseTimeout := 60 // minutes

	if rawConfig.CommandTimeout > 0 {
		commandTimeout := util.MinInt(rawConfig.CommandTimeout, MaxCommandTimeout)
		p.options.CommandTimeout = commandTimeout * 60 * 1000 // convert to milliseconds
		p.logger.Debugln("CommandTimeout set in config, new CommandTimeout:", commandTimeout)
	}

	if rawConfig.NoResponseTimeout > 0 {
		noResponseTimeout := util.MinInt(rawConfig.NoResponseTimeout, MaxNoResponseTimeout)
		p.options.NoResponseTimeout = noResponseTimeout * 60 * 1000 // convert to milliseconds
		p.logger.Debugln("NoReponseTimeout set in config, new NoReponseTimeout:", noResponseTimeout)
	}

	return rawConfig, string(werckerYaml), nil
}

// AddServices fetches and links the services to the base box.
func (p *Runner) AddServices(ctx context.Context, pipeline core.Pipeline, box core.Box) error {
	f := p.formatter
	timer := util.NewTimer()
	for _, service := range pipeline.Services() {
		timer.Reset()
		if _, err := service.Fetch(ctx, pipeline.Env()); err != nil {
			return err
		}

		box.AddService(service)
		if p.options.Verbose {
			p.logger.Printf(f.Success(fmt.Sprintf("Fetched %s", service.GetName()), timer.String()))
		}
		// TODO(mh): We want to make sure container is running fully before
		// allowing build steps to run. We may need custom steps which block
		// until service services are running.
	}
	return nil
}

// CopyCache copies the source into the HostPath
func (p *Runner) CopyCache() error {
	timer := util.NewTimer()
	f := p.formatter

	err := os.MkdirAll(p.options.CachePath(), 0755)
	if err != nil {
		return err
	}

	err = os.Symlink(p.options.CachePath(), p.options.HostPath("cache"))
	if err != nil {
		return err
	}
	if p.options.Verbose {
		p.logger.Printf(f.Success("Cache -> Staging Area", timer.String()))
	}

	if p.options.Verbose {
		p.logger.Printf(f.Success("Cache -> Staging Area", timer.String()))
	}
	return nil
}

// CopySource copies the source into the HostPath
func (p *Runner) CopySource() error {
	timer := util.NewTimer()
	f := p.formatter

	err := os.MkdirAll(p.options.HostPath(), 0755)
	if err != nil {
		return err
	}

	err = os.Symlink(p.ProjectDir(), p.options.HostPath("source"))
	if err != nil {
		return err
	}
	if p.options.Verbose {
		p.logger.Printf(f.Success("Source -> Staging Area", timer.String()))
	}
	return nil
}

// GetSession attaches to the container and returns a session.
func (p *Runner) GetSession(runnerContext context.Context, containerID string) (context.Context, *core.Session, error) {
	dockerTransport, err := dockerlocal.NewDockerTransport(p.options, p.dockerOptions, containerID)
	if err != nil {
		return nil, nil, err
	}
	sess := core.NewSession(p.options, dockerTransport)
	if err != nil {
		return nil, nil, err
	}
	sessionCtx, err := sess.Attach(runnerContext)
	if err != nil {
		return nil, nil, err
	}

	return sessionCtx, sess, nil
}

// GetPipeline returns a pipeline based on the "build" config section
func (p *Runner) GetPipeline(rawConfig *core.Config) (core.Pipeline, error) {
	return p.getPipeline(rawConfig, p.options, p.dockerOptions)
}

// RunnerShared holds on to the information we got from setting up our
// environment.
type RunnerShared struct {
	box         core.Box
	pipeline    core.Pipeline
	sess        *core.Session
	config      *core.Config
	sessionCtx  context.Context
	containerID string
}

// StartStep emits BuildStepStarted and returns a Finisher for the end event.
func (p *Runner) StartStep(ctx *RunnerShared, step core.Step, order int) *util.Finisher {
	p.emitter.Emit(core.BuildStepStarted, &core.BuildStepStartedArgs{
		Box:   ctx.box,
		Step:  step,
		Order: order,
	})
	return util.NewFinisher(func(result interface{}) {
		r := result.(*StepResult)
		artifactURL := ""
		if r.Artifact != nil {
			artifactURL = r.Artifact.URL()
		}
		p.emitter.Emit(core.BuildStepFinished, &core.BuildStepFinishedArgs{
			Box:                 ctx.box,
			Successful:          r.Success,
			Message:             r.Message,
			ArtifactURL:         artifactURL,
			PackageURL:          r.PackageURL,
			WerckerYamlContents: r.WerckerYamlContents,
		})
	})
}

// StartBuild emits a BuildStarted and returns for a Finisher for the end.
func (p *Runner) StartBuild(options *core.PipelineOptions) *util.Finisher {
	p.emitter.Emit(core.BuildStarted, &core.BuildStartedArgs{Options: options})
	return util.NewFinisher(func(result interface{}) {
		r, ok := result.(*core.BuildFinishedArgs)
		if !ok {
			return
		}
		r.Options = options
		p.emitter.Emit(core.BuildFinished, r)
	})
}

// StartFullPipeline emits a FullPipelineFinished when the Finisher is called.
func (p *Runner) StartFullPipeline(options *core.PipelineOptions) *util.Finisher {
	return util.NewFinisher(func(result interface{}) {
		r, ok := result.(*core.FullPipelineFinishedArgs)
		if !ok {
			return
		}

		r.Options = options
		p.emitter.Emit(core.FullPipelineFinished, r)
	})
}

// SetupEnvironment does a lot of boilerplate legwork and returns a pipeline,
// box, and session. This is a bit of a long method, but it is pretty much
// the entire "Setup Environment" step.
func (p *Runner) SetupEnvironment(runnerCtx context.Context) (*RunnerShared, error) {
	shared := &RunnerShared{}
	f := &util.Formatter{p.options.GlobalOptions.ShowColors}
	timer := util.NewTimer()

	sr := &StepResult{
		Success:  false,
		Artifact: nil,
		Message:  "",
		ExitCode: 1,
	}

	setupEnvironmentStep := &core.ExternalStep{
		BaseStep: core.NewBaseStep(core.BaseStepOptions{
			Name:    "setup environment",
			Owner:   "wercker",
			Version: util.Version(),
		}),
	}
	finisher := p.StartStep(shared, setupEnvironmentStep, 2)
	defer finisher.Finish(sr)

	if p.options.Verbose {
		p.emitter.Emit(core.Logs, &core.LogsArgs{
			Logs: fmt.Sprintf("Running wercker version: %s\n", util.FullVersion()),
		})
	}

	p.logger.Debugln("Application:", p.options.ApplicationName)

	// Grab our config
	rawConfig, stringConfig, err := p.GetConfig()
	if stringConfig != "" && p.options.Verbose {
		p.emitter.Emit(core.Logs, &core.LogsArgs{
			Logs: fmt.Sprintf("Using config:\n%s\n", stringConfig),
		})
	}
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}
	shared.config = rawConfig
	sr.WerckerYamlContents = stringConfig

	// Init the pipeline
	pipeline, err := p.GetPipeline(rawConfig)
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}
	pipeline.InitEnv(p.options.HostEnv)
	shared.pipeline = pipeline

	// Fetch the box
	timer.Reset()
	box := pipeline.Box()
	_, err = box.Fetch(runnerCtx, pipeline.Env())
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	// TODO(termie): dump some logs about the image
	shared.box = box
	if p.options.Verbose {
		p.logger.Printf(f.Success(fmt.Sprintf("Fetched %s", box.GetName()), timer.String()))
	}

	// Fetch the services and add them to the box
	if err := p.AddServices(runnerCtx, pipeline, box); err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	// Start setting up the pipeline dir
	p.logger.Debugln("Copying source to build directory")
	err = p.CopySource()
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	// ... and the cache dir
	p.logger.Debugln("Copying cache to build directory")
	err = p.CopyCache()
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	pipeline.LocalSymlink()

	p.logger.Debugln("Steps:", len(pipeline.Steps()))

	// Fetch the steps
	steps := pipeline.Steps()
	for _, step := range steps {
		timer.Reset()
		if _, err := step.Fetch(); err != nil {
			sr.Message = err.Error()
			return shared, err
		}
		if p.options.Verbose {
			p.logger.Printf(f.Success("Prepared step", step.Name(), timer.String()))
		}

	}

	// ... and the after steps
	afterSteps := pipeline.AfterSteps()
	for _, step := range afterSteps {
		timer.Reset()
		if _, err := step.Fetch(); err != nil {
			sr.Message = err.Error()
			return shared, err
		}

		if p.options.Verbose {
			p.logger.Printf(f.Success("Prepared step", step.Name(), timer.String()))
		}
	}

	// Boot up our main container, it will run the services
	container, err := box.Run(runnerCtx, pipeline.Env())
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}
	shared.containerID = container.ID

	// Register our signal handler to clean the box up
	// NOTE(termie): we're expecting that this is going to be the last handler
	//               to be run since it calls exit, in the future we might be
	//               able to do something like close the calling context and
	//               short circuit / let the rest of things play out
	boxCleanupHandler := &util.SignalHandler{
		ID: "box-cleanup",
		F: func() bool {
			p.logger.Errorln("Interrupt detected, cleaning up containers and shutting down")
			box.Stop()
			if p.options.ShouldRemove {
				box.Clean()
			}
			os.Exit(1)
			return true
		},
	}
	util.GlobalSigint().Add(boxCleanupHandler)
	util.GlobalSigterm().Add(boxCleanupHandler)

	p.logger.Debugln("Attaching session to base box")
	// Start our session
	sessionCtx, sess, err := p.GetSession(runnerCtx, container.ID)
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}
	shared.sess = sess
	shared.sessionCtx = sessionCtx

	// Some helpful logging
	pipeline.LogEnvironment()

	p.logger.Debugln("Setting up guest (base box)")
	err = pipeline.SetupGuest(sessionCtx, sess)
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	err = pipeline.ExportEnvironment(sessionCtx, sess)
	if err != nil {
		sr.Message = err.Error()
		return shared, err
	}

	sr.Message = ""
	sr.Success = true
	sr.ExitCode = 0
	return shared, nil
}

// StepResult holds the info we need to report on steps
type StepResult struct {
	Success             bool
	Artifact            *core.Artifact
	PackageURL          string
	Message             string
	ExitCode            int
	WerckerYamlContents string
}

// RunStep runs a step and tosses error if it fails
func (p *Runner) RunStep(shared *RunnerShared, step core.Step, order int) (*StepResult, error) {
	finisher := p.StartStep(shared, step, order)
	sr := &StepResult{
		Success:  false,
		Artifact: nil,
		Message:  "",
		ExitCode: 1,
	}
	defer finisher.Finish(sr)

	if step.ShouldSyncEnv() {
		err := shared.pipeline.SyncEnvironment(shared.sessionCtx, shared.sess)
		if err != nil {
			// If an error occured, just log and ignore it
			p.logger.WithField("Error", err).Warn("Unable to sync environment")
		}
	}

	step.InitEnv(shared.pipeline.Env())
	p.logger.Debugln("Step Environment")
	for _, pair := range step.Env().Ordered() {
		p.logger.Debugln(" ", pair[0], pair[1])
	}

	exit, err := step.Execute(shared.sessionCtx, shared.sess)
	if exit != 0 {
		sr.ExitCode = exit
		if p.options.AttachOnError {
			shared.box.RecoverInteractive(
				p.options.SourcePath(),
				shared.pipeline,
				step,
			)
		}
	} else if err == nil {
		sr.Success = true
		sr.ExitCode = 0
	}

	// Grab the message
	var message bytes.Buffer
	messageErr := step.CollectFile(shared.containerID, step.ReportPath(), "message.txt", &message)
	if messageErr != nil {
		if messageErr != util.ErrEmptyTarball {
			return sr, messageErr
		}
	}
	sr.Message = message.String()

	// This is the error from the step.Execute above
	if err != nil {
		if sr.Message == "" {
			sr.Message = err.Error()
		}
		return sr, err
	}

	// Grab artifacts if we want them
	if p.options.ShouldArtifacts {
		artifact, err := step.CollectArtifact(shared.containerID)
		if err != nil {
			return sr, err
		}

		if artifact != nil && p.options.ShouldStoreS3 {
			artificer := dockerlocal.NewArtificer(p.options, p.dockerOptions)
			err = artificer.Upload(artifact)
			if err != nil {
				return sr, err
			}
		}
		sr.Artifact = artifact
	}

	if !sr.Success {
		return sr, fmt.Errorf("Step failed with exit code: %d", sr.ExitCode)
	}

	return sr, nil
}
