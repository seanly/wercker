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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"github.com/joho/godotenv"
	"github.com/mreiferson/go-snappystream"
	"github.com/wercker/journalhook"
	"github.com/wercker/wercker/api"
	"github.com/wercker/wercker/core"
	"github.com/wercker/wercker/docker"
	"github.com/wercker/wercker/util"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

var (
	cliLogger    = util.RootLogger().WithField("Logger", "CLI")
	buildCommand = cli.Command{
		Name:      "build",
		ShortName: "b",
		Usage:     "build a project",
		Action: func(c *cli.Context) {
			envfile := c.GlobalString("environment")
			_ = godotenv.Load(envfile)

			env := util.NewEnvironment(os.Environ()...)

			settings := util.NewCLISettings(c)
			opts, err := core.NewBuildOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			_, err = cmdBuild(context.Background(), opts, dockerOptions)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
		Flags: FlagsFor(PipelineFlagSet, WerckerInternalFlagSet),
	}

	devCommand = cli.Command{
		Name:  "dev",
		Usage: "develop and run a local project",
		Action: func(c *cli.Context) {
			envfile := c.GlobalString("environment")
			_ = godotenv.Load(envfile)

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewDevOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			_, err = cmdDev(context.Background(), opts, dockerOptions)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
		Flags: FlagsFor(DevPipelineFlagSet, WerckerInternalFlagSet),
	}

	checkConfigCommand = cli.Command{
		Name: "check-config",
		// ShortName: "b",
		Usage: "check the project's yaml",
		Action: func(c *cli.Context) {
			envfile := c.GlobalString("environment")
			_ = godotenv.Load(envfile)

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewCheckConfigOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdCheckConfig(opts, dockerOptions)
			if err != nil {
				os.Exit(1)
			}
		},
		Flags: FlagsFor(PipelineFlagSet, WerckerInternalFlagSet),
	}

	deployCommand = cli.Command{
		Name:      "deploy",
		ShortName: "d",
		Usage:     "deploy a project",
		Action: func(c *cli.Context) {
			envfile := c.GlobalString("environment")
			_ = godotenv.Load(envfile)

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewDeployOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			_, err = cmdDeploy(context.Background(), opts, dockerOptions)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
		Flags: FlagsFor(DeployPipelineFlagSet, WerckerInternalFlagSet),
	}

	detectCommand = cli.Command{
		Name:      "detect",
		ShortName: "de",
		Usage:     "detect the type of project",
		Flags:     []cli.Flag{},
		Action: func(c *cli.Context) {
			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewDetectOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdDetect(opts)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
	}

	inspectCommand = cli.Command{
		Name:      "inspect",
		ShortName: "i",
		Usage:     "inspect a recent container",
		Action: func(c *cli.Context) {
			// envfile := c.GlobalString("environment")
			// _ = godotenv.Load(envfile)

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewInspectOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdInspect(opts, dockerOptions)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
		Flags: FlagsFor(PipelineFlagSet, WerckerInternalFlagSet),
	}

	loginCommand = cli.Command{
		Name:      "login",
		ShortName: "l",
		Usage:     "log into wercker",
		Flags:     []cli.Flag{},
		Action: func(c *cli.Context) {
			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewLoginOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdLogin(opts)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
	}

	logoutCommand = cli.Command{
		Name:  "logout",
		Usage: "logout from wercker",
		Flags: []cli.Flag{},
		Action: func(c *cli.Context) {

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewLogoutOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdLogout(opts)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
	}

	pullCommand = cli.Command{
		Name:        "pull",
		ShortName:   "p",
		Usage:       "pull <build id>",
		Description: "download a Docker repository, and load it into Docker",
		Flags:       FlagsFor(DockerFlagSet, PullFlagSet),
		Action: func(c *cli.Context) {
			if len(c.Args()) != 1 {
				cliLogger.Errorln("Pull requires the application ID or the build ID as the only argument")
				os.Exit(1)
			}

			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewPullOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			dockerOptions, err := dockerlocal.NewDockerOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdPull(c, opts, dockerOptions)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
	}

	versionCommand = cli.Command{
		Name:      "version",
		ShortName: "v",
		Usage:     "print versions",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "json",
				Usage: "Output version information as JSON",
			},
			cli.BoolFlag{
				Name:  "beta",
				Usage: "Checks for the latest beta version",
			},
			cli.BoolFlag{
				Name:  "no-update-check",
				Usage: "Do not check for update",
			},
		},
		Action: func(c *cli.Context) {
			settings := util.NewCLISettings(c)
			env := util.NewEnvironment(os.Environ()...)
			opts, err := core.NewVersionOptions(settings, env)
			if err != nil {
				cliLogger.Errorln("Invalid options\n", err)
				os.Exit(1)
			}
			err = cmdVersion(opts)
			if err != nil {
				cliLogger.Fatal(err)
			}
		},
	}

	documentCommand = func(app *cli.App) cli.Command {
		return cli.Command{
			Name:  "doc",
			Usage: "Generate usage documentation",
			Action: func(c *cli.Context) {
				settings := util.NewCLISettings(c)
				env := util.NewEnvironment(os.Environ()...)
				opts, err := core.NewGlobalOptions(settings, env)
				if err != nil {
					cliLogger.Errorln("Invalid options\n", err)
					os.Exit(1)
				}
				if err := GenerateDocumentation(opts, app); err != nil {
					cliLogger.Fatal(err)
				}
			},
		}
	}
)

func GetApp() *cli.App {
	// logger.SetLevel(logger.DebugLevel)
	// util.RootLogger().SetLevel("debug")
	// util.RootLogger().Formatter = &logger.JSONFormatter{}

	app := cli.NewApp()
	setupUsageFormatter(app)
	app.Author = "Team wercker"
	app.Name = "wercker"
	app.Usage = "build and deploy from the command line"
	app.Email = "pleasemailus@wercker.com"
	app.Version = util.FullVersion()
	app.Flags = FlagsFor(GlobalFlagSet)
	app.Commands = []cli.Command{
		buildCommand,
		devCommand,
		checkConfigCommand,
		deployCommand,
		detectCommand,
		// inspectCommand,
		loginCommand,
		logoutCommand,
		pullCommand,
		versionCommand,
		documentCommand(app),
	}
	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalBool("debug") {
			util.RootLogger().Formatter = &util.VerboseFormatter{}
			util.RootLogger().SetLevel("debug")
		} else {
			util.RootLogger().Formatter = &util.TerseFormatter{}
			util.RootLogger().SetLevel("info")
		}
		if ctx.GlobalBool("journal") {
			util.RootLogger().Hooks.Add(&journalhook.JournalHook{})
			util.RootLogger().Out = ioutil.Discard
		}
		// Register the global signal handler
		util.GlobalSigint().Register(os.Interrupt)
		util.GlobalSigterm().Register(unix.SIGTERM)
		return nil
	}
	return app
}

// SoftExit is a helper for determining when to show stack traces
type SoftExit struct {
	options *core.GlobalOptions
}

// NewSoftExit constructor
func NewSoftExit(options *core.GlobalOptions) *SoftExit {
	return &SoftExit{options}
}

// Exit with either an error or a panic
func (s *SoftExit) Exit(v ...interface{}) error {
	if s.options.Debug {
		// Clearly this will cause it's own exit if it gets called.
		util.RootLogger().Panicln(v...)
	}
	util.RootLogger().Errorln(v...)
	return fmt.Errorf("Exiting.")
}

func cmdDev(ctx context.Context, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (*RunnerShared, error) {
	if options.Pipeline == "" {
		options.Pipeline = "dev"
	}
	pipelineGetter := GetDevPipelineFactory(options.Pipeline)
	ctx = core.NewEmitterContext(ctx)
	return executePipeline(ctx, options, dockerOptions, pipelineGetter)
}

func cmdBuild(ctx context.Context, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (*RunnerShared, error) {
	if options.Pipeline == "" {
		options.Pipeline = "build"
	}
	pipelineGetter := GetBuildPipelineFactory(options.Pipeline)
	ctx = core.NewEmitterContext(ctx)
	return executePipeline(ctx, options, dockerOptions, pipelineGetter)
}

func cmdDeploy(ctx context.Context, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) (*RunnerShared, error) {
	if options.Pipeline == "" {
		options.Pipeline = "deploy"
	}
	pipelineGetter := GetDeployPipelineFactory(options.Pipeline)
	ctx = core.NewEmitterContext(ctx)
	return executePipeline(ctx, options, dockerOptions, pipelineGetter)
}

func cmdCheckConfig(options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions) error {
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")

	// TODO(termie): this is pretty much copy-paste from the
	//               runner.GetConfig step, we should probably refactor
	var werckerYaml []byte
	var err error
	if options.WerckerYml != "" {
		werckerYaml, err = ioutil.ReadFile(options.WerckerYml)
		if err != nil {
			return soft.Exit(err)
		}
	} else {
		werckerYaml, err = core.ReadWerckerYaml([]string{"."}, false)
		if err != nil {
			return soft.Exit(err)
		}
	}

	// Parse that bad boy.
	rawConfig, err := core.ConfigFromYaml(werckerYaml)
	if err != nil {
		return soft.Exit(err)
	}

	for name, _ := range rawConfig.PipelinesMap {
		build, err := dockerlocal.NewDockerPipeline(name, rawConfig, options, dockerOptions, dockerlocal.NewNilBuilder())
		if err != nil {
			return soft.Exit(err)
		}
		logger.Println("Found pipeline section:", name)
		if build.Box() != nil {
			logger.Println("  with box:", build.Box().GetName())
		}
	}

	return nil
}

// detectProject inspects the the current directory that wercker is running in
// and detects the project's programming language
func cmdDetect(options *core.DetectOptions) error {
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")

	logger.Println("########### Detecting your project! #############")

	detected := ""

	d, err := os.Open(".")
	if err != nil {
		logger.WithField("Error", err).Error("Unable to open directory")
		soft.Exit(err)
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to read directory")
		soft.Exit(err)
	}
outer:
	for _, f := range files {
		switch {
		case f.Name() == "package.json":
			detected = "nodejs"
			break outer

		case f.Name() == "requirements.txt":
			detected = "python"
			break outer

		case f.Name() == "Gemfile":
			detected = "ruby"
			break outer

		case filepath.Ext(f.Name()) == ".go":
			detected = "golang"
			break outer
		}
	}
	if detected == "" {
		logger.Println("No stack detected, generating default wercker.yml")
		detected = "default"
	} else {
		logger.Println("Detected:", detected)
		logger.Println("Generating wercker.yml")
	}
	getYml(detected, options)
	return nil
}

func cmdInspect(options *core.InspectOptions, dockerOptions *dockerlocal.DockerOptions) error {
	repoName := fmt.Sprintf("%s/%s", options.ApplicationOwnerName, options.ApplicationName)
	tag := options.Tag

	client, err := dockerlocal.NewDockerClient(dockerOptions)
	if err != nil {
		return err
	}

	return client.RunAndAttach(fmt.Sprintf("%s:%s", repoName, tag))
}

func cmdLogin(options *core.LoginOptions) error {
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")

	logger.Println("########### Logging into wercker! #############")
	url := fmt.Sprintf("%s/api/1.0/%s", options.BaseURL, "oauth/basicauthaccesstoken")

	username := readUsername()
	password := readPassword()

	token, err := getAccessToken(username, password, url)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to log into wercker")
		return soft.Exit(err)
	}

	logger.Println("Saving token to: ", options.AuthTokenStore)
	return saveToken(options.AuthTokenStore, token)
}

func cmdLogout(options *core.LogoutOptions) error {
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")

	logger.Println("Logging out")

	err := removeToken(options.GlobalOptions.AuthTokenStore)
	if err != nil {
		return soft.Exit(err)
	}
	return nil
}

func cmdPull(c *cli.Context, options *core.PullOptions, dockerOptions *dockerlocal.DockerOptions) error {
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")

	if options.Debug {
		DumpOptions(options)
	}

	client := api.NewAPIClient(&api.APIOptions{
		BaseURL:   options.GlobalOptions.BaseURL,
		AuthToken: options.GlobalOptions.AuthToken,
	})

	var buildID string

	if core.IsBuildID(options.Repository) {
		buildID = options.Repository
	} else {
		username, applicationName, err := core.ParseApplicationID(options.Repository)
		if err != nil {
			return soft.Exit(err)
		}

		logger.Println("Fetching build information for application", options.Repository)

		opts := &api.GetBuildsOptions{
			Limit:  1,
			Branch: options.Branch,
			Result: options.Result,
			Status: "finished",
			Stack:  5,
		}

		builds, err := client.GetBuilds(username, applicationName, opts)
		if err != nil {
			return soft.Exit(err)
		}

		if len(builds) != 1 {
			return soft.Exit(errors.New("No finished builds found for this application"))
		}

		buildID = builds[0].ID
	}

	if buildID == "" {
		return soft.Exit(errors.New("Unable to parse argument as application or build-id"))
	}

	logger.Println("Downloading Docker repository for build", buildID)

	if !options.Force {
		outputExists, err := util.Exists(options.Output)
		if err != nil {
			logger.WithField("Error", err).Error("Unable to create output file")
			return soft.Exit(err)
		}

		if outputExists {
			return soft.Exit(errors.New("The file repository.tar already exists. Delete it, or run again with -f"))
		}
	}

	file, err := os.Create(options.Output)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to create output file")
		return soft.Exit(err)
	}

	repository, err := client.GetDockerRepository(buildID)
	if err != nil {
		os.Remove(file.Name())
		return soft.Exit(err)
	}
	defer repository.Content.Close()

	// Diagram of the various readers/writers
	//   repository <-- tee <-- s <-- [io.Copy] --> file
	//               |
	//               +--> hash       *Legend: --> == write, <-- == read

	counter := util.NewCounterReader(repository.Content)

	stopEmit := emitProgress(counter, repository.Size, util.NewRawLogger())

	hash := sha256.New()
	tee := io.TeeReader(counter, hash)
	s := snappystream.NewReader(tee, true)

	_, err = io.Copy(file, s)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to copy data from URL to file")
		os.Remove(file.Name())
		return soft.Exit(err)
	}

	stopEmit <- true

	logger.Println("Download complete")

	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if calculatedHash != repository.Sha256 {
		return soft.Exit(fmt.Errorf("Calculated hash did not match provided hash (calculated: %s ; expected: %s)", calculatedHash, repository.Sha256))
	}

	if options.Load {
		_, err = file.Seek(0, 0)
		if err != nil {
			logger.WithField("Error", err).Error("Unable to reset seeker")
			return soft.Exit(err)
		}

		dockerClient, err := dockerlocal.NewDockerClient(dockerOptions)
		if err != nil {
			logger.WithField("Error", err).Error("Unable to create Docker client")
			return soft.Exit(err)
		}

		logger.Println("Importing into Docker")

		importImageOptions := docker.LoadImageOptions{InputStream: file}
		err = dockerClient.LoadImage(importImageOptions)
		if err != nil {
			logger.WithField("Error", err).Error("Unable to load image")
			return soft.Exit(err)
		}

		logger.Println("Finished importing into Docker")
	}

	return nil
}

// Retrieving user input utility functions
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		util.RootLogger().WithField("Logger", "Util").Fatal(err)
	}
	response = strings.ToLower(response)
	if strings.HasPrefix(response, "y") {
		return true
	} else if strings.HasPrefix(response, "n") {
		return false
	} else {
		println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

// emitProgress will keep emitting progress until a value is send into the
// returned channel.
func emitProgress(counter *util.CounterReader, total int64, logger *util.Logger) chan<- bool {
	stop := make(chan bool)
	go func(stop chan bool, counter *util.CounterReader, total int64) {
		prev := int64(-1)
		for {
			current := counter.Count()
			percentage := (100 * current) / total

			select {
			case <-stop:
				logger.Infof("\rDownloading: %3d%%\n", percentage)
				return
			default:
				if percentage != prev {
					logger.Infof("\rDownloading: %3d%%", percentage)
					prev = percentage
				}
				time.Sleep(1 * time.Second)
			}
		}
	}(stop, counter, total)
	return stop
}

func cmdVersion(options *core.VersionOptions) error {
	logger := util.RootLogger().WithField("Logger", "Main")

	v := util.GetVersions()

	if options.OutputJSON {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			logger.WithField("Error", err).Panic("Unable to marshal versions")
		}
		os.Stdout.Write(b)
		os.Stdout.WriteString("\n")
		return nil
	}

	logger.Infoln("Version:", v.Version)
	logger.Infoln("Compiled at:", v.CompiledAt.Local())

	if v.GitCommit != "" {
		logger.Infoln("Git commit:", v.GitCommit)
	}

	if options.CheckForUpdate {
		channel := "stable"
		if options.BetaChannel {
			channel = "beta"
		}
		updater, err := NewUpdater(channel)
		if err != nil {
			return err
		}
		if updater.UpdateAvailable() {
			logger.Infoln("A new version is available:",
				updater.ServerVersion.FullVersion())

			// try to determine if binary was installed with homebrew
			homebrew := util.InstalledWithHomebrew()

			if homebrew {
				logger.Info("\nLooks like wercker was installed with homebrew.\n\n" +
					"To update to the latest version please use:\n" +
					"brew update && brew upgrade wercker-cli")

				logger.Println("\nUsing the built in updater can cause issues with Wercker installed with homebrew.")
			} else {
				logger.Infoln("Download it from:", updater.DownloadURL())
			}

			if AskForUpdate() {
				if err := updater.Update(); err != nil {
					logger.WithField("Error", err).Warn(
						"Unable to download latest version. Please try again.")
					return err
				}
			}
		} else {
			logger.Infoln("No new version available")
		}
	}
	return nil
}

// TODO(mies): maybe move to util.go at some point
func getYml(detected string, options *core.DetectOptions) {
	logger := util.RootLogger().WithField("Logger", "Main")

	yml := "wercker.yml"
	if _, err := os.Stat(yml); err == nil {
		logger.Println(yml, "already exists. Do you want to overwrite? (yes/no)")
		if !askForConfirmation() {
			logger.Println("Exiting...")
			os.Exit(1)
		}
	}
	url := fmt.Sprintf("%s/api/v2/yml/%s", options.BaseURL, detected)
	res, err := http.Get(url)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to reach wercker API")
		os.Exit(1)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to read response")
	}

	err = ioutil.WriteFile("wercker.yml", body, 0644)
	if err != nil {
		logger.WithField("Error", err).Error("Unable to write wercker.yml file")
	}

}

// DumpOptions prints out a sorted list of options
func DumpOptions(options interface{}, indent ...string) {
	indent = append(indent, "  ")
	s := reflect.ValueOf(options).Elem()
	typeOfT := s.Type()
	names := []string{}
	for i := 0; i < s.NumField(); i++ {
		// f := s.Field(i)
		fieldName := typeOfT.Field(i).Name
		if fieldName != "HostEnv" {
			names = append(names, fieldName)
		}
	}
	sort.Strings(names)
	logger := util.RootLogger().WithField("Logger", "Options")

	for _, name := range names {
		r := reflect.ValueOf(options)
		f := reflect.Indirect(r).FieldByName(name)
		if strings.HasSuffix(name, "Options") {
			if len(indent) > 1 && name == "GlobalOptions" {
				continue
			}
			logger.Debugln(fmt.Sprintf("%s%s %s", strings.Join(indent, ""), name, f.Type()))
			DumpOptions(f.Interface(), indent...)
		} else {
			logger.Debugln(fmt.Sprintf("%s%s %s = %v", strings.Join(indent, ""), name, f.Type(), f.Interface()))
		}
	}
}

func executePipeline(cmdCtx context.Context, options *core.PipelineOptions, dockerOptions *dockerlocal.DockerOptions, getter pipelineGetter) (*RunnerShared, error) {
	// Boilerplate
	soft := NewSoftExit(options.GlobalOptions)
	logger := util.RootLogger().WithField("Logger", "Main")
	e, err := core.EmitterFromContext(cmdCtx)
	if err != nil {
		return nil, err
	}
	f := &util.Formatter{options.GlobalOptions.ShowColors}

	// Set up the runner
	r, err := NewRunner(cmdCtx, options, dockerOptions, getter)
	if err != nil {
		return nil, err
	}

	// Main timer
	mainTimer := util.NewTimer()
	timer := util.NewTimer()

	// These will be emitted at the end of the execution, we're going to be
	// pessimistic and report that we failed, unless overridden at the end of the
	// execution.
	fullPipelineFinisher := r.StartFullPipeline(options)
	pipelineArgs := &core.FullPipelineFinishedArgs{}
	defer fullPipelineFinisher.Finish(pipelineArgs)

	buildFinisher := r.StartBuild(options)
	buildFinishedArgs := &core.BuildFinishedArgs{Box: nil, Result: "failed"}
	defer buildFinisher.Finish(buildFinishedArgs)

	// Debug information
	DumpOptions(options)

	// Do some sanity checks before starting
	err = dockerlocal.RequireDockerEndpoint(dockerOptions)
	if err != nil {
		return nil, soft.Exit(err)
	}

	// Make sure that "include-file" is read from the config file before copying code
	r.GetConfig()

	// Start copying code
	logger.Println(f.Info("Executing pipeline"))
	timer.Reset()
	_, err = r.EnsureCode()
	if err != nil {
		e.Emit(core.Logs, &core.LogsArgs{
			Stream: "stderr",
			Logs:   err.Error() + "\n",
		})
		return nil, soft.Exit(err)
	}
	err = r.CleanupOldBuilds()
	if err != nil {
		e.Emit(core.Logs, &core.LogsArgs{
			Stream: "stderr",
			Logs:   err.Error() + "\n",
		})
	}
	if options.Verbose {
		logger.Printf(f.Success("Copied working dir", timer.String()))
	}

	// Setup environment is still a fairly special step, it needs
	// to start our boxes and get everything set up
	logger.Println(f.Info("Running step", "setup environment"))
	timer.Reset()
	shared, err := r.SetupEnvironment(cmdCtx)
	if shared.box != nil {
		if options.ShouldRemove {
			defer shared.box.Clean()
		}
		defer shared.box.Stop()
	}
	if err != nil {
		logger.Errorln(f.Fail("Step failed", "setup environment", timer.String()))
		e.Emit(core.Logs, &core.LogsArgs{
			Stream: "stderr",
			Logs:   err.Error() + "\n",
		})
		return nil, soft.Exit(err)
	}
	if options.Verbose {
		logger.Printf(f.Success("Step passed", "setup environment", timer.String()))
	}

	// Expand our context object
	box := shared.box
	buildFinishedArgs.Box = box
	pipeline := shared.pipeline
	repoName := pipeline.DockerRepo()
	tag := pipeline.DockerTag()
	message := pipeline.DockerMessage()

	shouldStore := options.ShouldArtifacts

	// TODO(termie): hack for now, probably can be made into a naive class
	var storeStep core.Step

	if shouldStore {
		storeStep = &core.ExternalStep{
			BaseStep: core.NewBaseStep(core.BaseStepOptions{
				Name:    "store",
				Owner:   "wercker",
				Version: util.Version(),
			}),
		}
	}

	e.Emit(core.BuildStepsAdded, &core.BuildStepsAddedArgs{
		Build:      pipeline,
		Steps:      pipeline.Steps(),
		StoreStep:  storeStep,
		AfterSteps: pipeline.AfterSteps(),
	})

	pr := &core.PipelineResult{
		Success:           true,
		FailedStepName:    "",
		FailedStepMessage: "",
	}

	// stepCounter starts at 3, step 1 is "get code", step 2 is "setup
	// environment".
	stepCounter := &util.Counter{Current: 3}
	checkpoint := false
	for _, step := range pipeline.Steps() {
		// we always want to run the wercker-init step to provide some functions
		if !checkpoint && stepCounter.Current > 3 {
			if options.EnableDevSteps && options.Checkpoint != "" {
				logger.Printf(f.Info("Skipping step", step.DisplayName()))
				// start at the one after the checkpoint
				if step.Checkpoint() == options.Checkpoint {
					logger.Printf(f.Info("Found checkpoint", options.Checkpoint))
					checkpoint = true
				}
				stepCounter.Increment()
				continue
			}
		}
		logger.Printf(f.Info("Running step", step.DisplayName()))
		timer.Reset()
		sr, err := r.RunStep(shared, step, stepCounter.Increment())
		if err != nil {
			pr.Success = false
			pr.FailedStepName = step.DisplayName()
			pr.FailedStepMessage = sr.Message
			logger.Printf(f.Fail("Step failed", step.DisplayName(), timer.String()))
			break
		}

		if options.EnableDevSteps && step.Checkpoint() != "" {
			logger.Printf(f.Info("Checkpointing", step.Checkpoint()))
			box.Commit(box.Repository(), fmt.Sprintf("w-%s", step.Checkpoint()), "checkpoint", false)
		}

		if options.Verbose {
			logger.Printf(f.Success("Step passed", step.DisplayName(), timer.String()))
		}
	}

	if options.ShouldCommit {
		_, err = box.Commit(repoName, tag, message, true)
		if err != nil {
			logger.Errorln("Failed to commit:", err.Error())
		}
	}

	// We need to wind the counter to where it should be if we failed a step
	// so that is the number of steps + get code + setup environment + store
	// TODO(termie): remove all the this "order" stuff completely
	stepCounter.Current = len(pipeline.Steps()) + 3

	if pr.Success && options.ShouldArtifacts {
		// At this point the build has effectively passed but we can still mess it
		// up by being unable to deliver the artifacts

		err = func() error {
			sr := &StepResult{
				Success:    false,
				Artifact:   nil,
				Message:    "",
				PackageURL: "",
				ExitCode:   1,
			}
			finisher := r.StartStep(shared, storeStep, stepCounter.Increment())
			defer finisher.Finish(sr)

			pr.FailedStepName = storeStep.Name()
			pr.FailedStepMessage = "Unable to store pipeline output"

			e.Emit(core.Logs, &core.LogsArgs{
				Logs: "Storing artifacts\n",
			})

			artifact, err := pipeline.CollectArtifact(shared.containerID)
			// Ignore ErrEmptyTarball errors
			if err != util.ErrEmptyTarball {
				if err != nil {
					sr.Message = err.Error()
					e.Emit(core.Logs, &core.LogsArgs{
						Logs: fmt.Sprintf("Storing artifacts failed: %s\n", sr.Message),
					})
					return err
				}

				e.Emit(core.Logs, &core.LogsArgs{
					Logs: fmt.Sprintf("Collecting files from %s\n", artifact.GuestPath),
				})

				ignoredDirectories := []string{".git", "node_modules", "vendor", "site-packages"}
				nameEmit := func(path string, info os.FileInfo, err error) error {
					relativePath := strings.TrimPrefix(path, artifact.HostPath)

					if info.IsDir() {
						if util.ContainsString(ignoredDirectories, info.Name()) {
							e.Emit(core.Logs, &core.LogsArgs{
								Logs: fmt.Sprintf(".%s/ (content omitted)\n", relativePath),
							})
							return filepath.SkipDir
						}

						return nil
					}

					e.Emit(core.Logs, &core.LogsArgs{
						Logs: fmt.Sprintf(".%s\n", relativePath),
					})

					return nil
				}

				err = filepath.Walk(artifact.HostPath, nameEmit)
				if err != nil {
					sr.Message = err.Error()
					e.Emit(core.Logs, &core.LogsArgs{
						Logs: fmt.Sprintf("Storing artifacts failed: %s\n", sr.Message),
					})
					return err
				}

				tarInfo, err := os.Stat(artifact.HostTarPath)
				if err != nil {
					if os.IsNotExist(err) {
						e.Emit(core.Logs, &core.LogsArgs{
							Logs: "No artifacts stored",
						})
					} else {
						sr.Message = err.Error()
						e.Emit(core.Logs, &core.LogsArgs{
							Logs: fmt.Sprintf("Storing artifacts failed: %s\n", sr.Message),
						})
						return err
					}
				} else {
					size, unit := util.ConvertUnit(tarInfo.Size())
					e.Emit(core.Logs, &core.LogsArgs{
						Logs: fmt.Sprintf("Total artifact size: %d %s\n", size, unit),
					})
				}

				if options.ShouldStoreS3 {
					artificer := dockerlocal.NewArtificer(options, dockerOptions)
					err = artificer.Upload(artifact)
					if err != nil {
						sr.Message = err.Error()
						e.Emit(core.Logs, &core.LogsArgs{
							Logs: fmt.Sprintf("Storing artifacts failed: %s\n", sr.Message),
						})
						return err
					}
				}

				sr.PackageURL = artifact.URL()
			} else {
				e.Emit(core.Logs, &core.LogsArgs{
					Logs: "No artifacts found\n",
				})
			}

			e.Emit(core.Logs, &core.LogsArgs{
				Logs: "Storing artifacts complete\n",
			})

			sr.Success = true
			sr.ExitCode = 0

			return nil
		}()
		if err != nil {
			pr.Success = false
			logger.WithField("Error", err).Error("Unable to store pipeline output")
		}
	} else {
		stepCounter.Increment()
	}

	// We're sending our build finished but we're not done yet,
	// now is time to run after-steps if we have any
	if pr.Success {
		logger.Println(f.Success("Steps passed", mainTimer.String()))
		buildFinishedArgs.Result = "passed"
	}
	buildFinisher.Finish(buildFinishedArgs)
	pipelineArgs.MainSuccessful = pr.Success

	if len(pipeline.AfterSteps()) == 0 {
		// We're about to end the build, so pull the cache and explode it
		// into the CacheDir
		if !options.DirectMount {
			timer.Reset()
			err = pipeline.CollectCache(shared.containerID)
			if err != nil {
				logger.WithField("Error", err).Error("Unable to store cache")
			}
			if options.Verbose {
				logger.Printf(f.Success("Exported Cache", timer.String()))
			}
		}

		if pr.Success {
			logger.Println(f.Success("Pipeline finished", mainTimer.String()))
		} else {
			logger.Println(f.Fail("Pipeline failed", mainTimer.String()))
		}

		if !pr.Success {
			return nil, fmt.Errorf("Step failed: %s", pr.FailedStepName)
		}
		return shared, nil
	}

	pipelineArgs.RanAfterSteps = true

	logger.Println(f.Info("Starting after-steps"))
	// The container may have died, either way we'll have a fresh env
	container, err := box.Restart()
	if err != nil {
		logger.Panicln(err)
	}

	newSessCtx, newSess, err := r.GetSession(cmdCtx, container.ID)
	if err != nil {
		logger.Panicln(err)
	}

	newShared := &RunnerShared{
		box:         shared.box,
		pipeline:    shared.pipeline,
		sess:        newSess,
		sessionCtx:  newSessCtx,
		containerID: shared.containerID,
		config:      shared.config,
	}

	// Set up the base environment
	err = pipeline.ExportEnvironment(newSessCtx, newSess)
	if err != nil {
		return nil, err
	}

	// Add the After-Step parts
	err = pr.ExportEnvironment(newSessCtx, newSess)
	if err != nil {
		return nil, err
	}

	for _, step := range pipeline.AfterSteps() {
		logger.Println(f.Info("Running after-step", step.DisplayName()))
		timer.Reset()
		_, err := r.RunStep(newShared, step, stepCounter.Increment())
		if err != nil {
			logger.Println(f.Fail("After-step failed", step.DisplayName(), timer.String()))
			break
		}
		logger.Println(f.Success("After-step passed", step.DisplayName(), timer.String()))
	}

	// We're about to end the build, so pull the cache and explode it
	// into the CacheDir
	if !options.DirectMount {
		timer.Reset()
		err = pipeline.CollectCache(newShared.containerID)
		if err != nil {
			logger.WithField("Error", err).Error("Unable to store cache")
		}
		if options.Verbose {
			logger.Printf(f.Success("Exported Cache", timer.String()))
		}
	}

	if pr.Success {
		logger.Println(f.Success("Pipeline finished", mainTimer.String()))
	} else {
		logger.Println(f.Fail("Pipeline failed", mainTimer.String()))
	}

	if !pr.Success {
		return nil, fmt.Errorf("Step failed: %s", pr.FailedStepName)
	}

	pipelineArgs.AfterStepSuccessful = pr.Success

	return shared, nil
}
