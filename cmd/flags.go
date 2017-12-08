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
	"github.com/codegangsta/cli"
	"github.com/wercker/wercker/core"
)

// Flags for setting these options from the CLI
var (
	// These flags tell us where to go for operations
	EndpointFlags = []cli.Flag{
		// deprecated
		cli.StringFlag{Name: "wercker-endpoint", Value: "", Usage: "Deprecated.", Hidden: true},
		cli.StringFlag{Name: "base-url", Value: core.DEFAULT_BASE_URL, Usage: "Base url for the wercker app.", Hidden: true},
		cli.StringFlag{Name: "steps-registry", Value: "https://steps.wercker.com", EnvVar: "STEPS_REGISTRY", Usage: "Endpoint for the steps registry", Hidden: true},
	}

	// These flags let us auth to wercker services
	AuthFlags = []cli.Flag{
		cli.StringFlag{Name: "auth-token", Usage: "Authentication token to use."},
		cli.StringFlag{Name: "auth-token-store", Value: "~/.wercker/token", Usage: "Where to store the token after a login.", Hidden: true},
	}

	DockerFlags = []cli.Flag{
		cli.StringFlag{Name: "docker-host", Value: "", Usage: "Docker api endpoint.", EnvVar: "DOCKER_HOST"},
		cli.StringFlag{Name: "docker-tls-verify", Value: "0", Usage: "Docker api tls verify.", EnvVar: "DOCKER_TLS_VERIFY"},
		cli.StringFlag{Name: "docker-cert-path", Value: "", Usage: "Docker api cert path.", EnvVar: "DOCKER_CERT_PATH"},
		cli.StringSliceFlag{Name: "docker-dns", Value: &cli.StringSlice{}, Usage: "Docker DNS server.", EnvVar: "DOCKER_DNS", Hidden: true},
		cli.BoolFlag{Name: "docker-local", Usage: "Don't interact with remote repositories"},
		cli.StringFlag{Name: "checkpoint", Value: "", Usage: "Skip to the next step after a recent build checkpoint."},
		cli.IntFlag{Name: "docker-cpu-period", Usage: "Set docker CPU period NOTIMPLEMENTED", Hidden: true},
		cli.IntFlag{Name: "docker-cpu-quota", Usage: "Set docker CPU quota NOTIMPLEMENTED", Hidden: true},
		cli.IntFlag{Name: "docker-memory", Usage: "Set docker user memory limit in MB", Hidden: true},
		cli.IntFlag{Name: "docker-memory-swap", Usage: "Set docker user memory swap limit in MB", Hidden: true},
		cli.IntFlag{Name: "docker-memory-reservation", Usage: "Set docker user memory soft limit in MB NOTIMPLEMENTED", Hidden: true},
		cli.IntFlag{Name: "docker-kernel-memory", Usage: "Set docker kernel memory limit in MB NOTIMPLEMENTED", Hidden: true},
		cli.BoolFlag{Name: "docker-cleanup-image", Usage: "Remove image from the Docker when finished pushing them", Hidden: true},
	}

	// These flags control where we store local files
	LocalPathFlags = []cli.Flag{
		cli.StringFlag{Name: "working-dir", Value: "./.wercker", Usage: "Path where we store working files.", EnvVar: "WERCKER_WORKING_DIR"},
	}

	// These flags control paths on the guest and probably shouldn't change
	InternalPathFlags = []cli.Flag{
		cli.StringFlag{Name: "mnt-root", Value: "/mnt", Usage: "Directory on the guest where volumes are mounted.", Hidden: true},
		cli.StringFlag{Name: "guest-root", Value: "/pipeline", Usage: "Directory on the guest where work is done.", Hidden: true},
		cli.StringFlag{Name: "report-root", Value: "/report", Usage: "Directory on the guest where reports will be written.", Hidden: true},
	}

	// These flags are usually pulled from the env
	WerckerFlags = []cli.Flag{
		cli.StringFlag{Name: "build-id", Value: "", EnvVar: "WERCKER_BUILD_ID", Hidden: true, Usage: "The build id (deprecated)."},
		cli.StringFlag{Name: "deploy-id", Value: "", EnvVar: "WERCKER_DEPLOY_ID", Hidden: true, Usage: "The deploy id (deprecated)."},
		cli.StringFlag{Name: "run-id", Value: "", EnvVar: "WERCKER_DEPLOY_ID", Hidden: true, Usage: "The run id."},
		cli.StringFlag{Name: "deploy-target", Value: "", EnvVar: "WERCKER_DEPLOYTARGET_NAME", Usage: "The deploy target name."},
		cli.StringFlag{Name: "application-id", Value: "", EnvVar: "WERCKER_APPLICATION_ID", Hidden: true, Usage: "The application id."},
		cli.StringFlag{Name: "application-name", Value: "", EnvVar: "WERCKER_APPLICATION_NAME", Hidden: true, Usage: "The application name."},
		cli.StringFlag{Name: "application-owner-name", Value: "", EnvVar: "WERCKER_APPLICATION_OWNER_NAME", Hidden: true, Usage: "The application owner name."},
		cli.StringFlag{Name: "application-started-by-name", Value: "", EnvVar: "WERCKER_APPLICATION_STARTED_BY_NAME", Hidden: true, Usage: "The name of the user who started the application."},
		cli.StringFlag{Name: "wercker-container-registry", Value: "https://wcr.io/v2/", Usage: "Wercker Registry Domain.", EnvVar: "WERCKER_CONTAINER_REGISTRY", Hidden: true},
		cli.StringFlag{Name: "pipeline", Value: "", EnvVar: "WERCKER_PIPELINE", Usage: "Alternate pipeline name to execute."},
	}

	GitFlags = []cli.Flag{
		cli.StringFlag{Name: "git-domain", Value: "", Usage: "Git domain.", EnvVar: "WERCKER_GIT_DOMAIN", Hidden: true},
		cli.StringFlag{Name: "git-owner", Value: "", Usage: "Git owner.", EnvVar: "WERCKER_GIT_OWNER", Hidden: true},
		cli.StringFlag{Name: "git-repository", Value: "", Usage: "Git repository.", EnvVar: "WERCKER_GIT_REPOSITORY", Hidden: true},
		cli.StringFlag{Name: "git-branch", Value: "", Usage: "Git branch.", EnvVar: "WERCKER_GIT_BRANCH", Hidden: true},
		cli.StringFlag{Name: "git-commit", Value: "", Usage: "Git commit.", EnvVar: "WERCKER_GIT_COMMIT", Hidden: true},
	}

	// These flags affect our registry interactions
	RegistryFlags = []cli.Flag{
		cli.StringFlag{Name: "commit", Value: "", Usage: "Commit the build result locally."},
		cli.StringFlag{Name: "tag", Value: "", Usage: "Tag for this build.", EnvVar: "WERCKER_GIT_BRANCH"},
		cli.StringFlag{Name: "message", Value: "", Usage: "Message for this build."},
	}

	// These flags affect our artifact interactions
	ArtifactFlags = []cli.Flag{
		cli.BoolFlag{Name: "artifacts", Usage: "Store artifacts."},
		cli.BoolFlag{Name: "no-remove", Usage: "Don't remove the containers."},
		cli.BoolFlag{Name: "store-s3",
			Usage: `Store artifacts and containers on s3.
			This requires access to aws credentials, pulled from any of the usual places
			(~/.aws/config, AWS_SECRET_ACCESS_KEY, etc), or from the --aws-secret-key and
			--aws-access-key flags. It will upload to a bucket defined by --s3-bucket in
			the region named by --aws-region`},
	}

	// These flags affect our local execution environment
	DevFlags = []cli.Flag{
		cli.StringFlag{Name: "environment", Value: "ENVIRONMENT", Usage: "Specify additional environment variables in a file.", EnvVar: "WERCKER_ENVIRONMENT_FILE"},
		cli.BoolFlag{Name: "verbose", Usage: "Print more information."},
		cli.BoolFlag{Name: "no-colors", Usage: "Wercker output will not use colors (does not apply to step output)."},
		cli.BoolFlag{Name: "debug", Usage: "Print additional debug information."},
		cli.BoolFlag{Name: "journal", Usage: "Send logs to systemd-journald. Suppresses stdout logging."},
	}

	// These flags are advanced dev settings
	InternalDevFlags = []cli.Flag{
		cli.BoolTFlag{Name: "direct-mount", Usage: "Mount our binds read-write to the pipeline path."},
		cli.BoolFlag{Name: "expose-ports", Usage: "Enable ports from wercker.yml beeing exposed to the host system."},
		// deprecated
		cli.StringSliceFlag{Name: "publish", Value: &cli.StringSlice{}, Usage: "[Deprecated] Use: --expose-ports. - Publish a port from the main container, same format as docker --publish.", Hidden: true},
		cli.BoolFlag{Name: "attach-on-error", Usage: "Attach shell to container if a step fails.", Hidden: true},
		cli.BoolFlag{Name: "enable-volumes", Usage: "Mount local files and directories as volumes to your wercker container, specified in your wercker.yml."},
		cli.BoolTFlag{Name: "enable-dev-steps", Hidden: true, Usage: `
		Enable internal dev steps.
		This enables:
		- internal/watch
		`},
	}

	// These flags are advanced build settings
	InternalBuildFlags = []cli.Flag{
		cli.BoolFlag{Name: "direct-mount", Usage: "Mount our binds read-write to the pipeline path."},
		cli.BoolFlag{Name: "expose-ports", Usage: "Enable ports from wercker.yml beeing exposed to the host system."},
		// deprecated
		cli.StringSliceFlag{Name: "publish", Value: &cli.StringSlice{}, Usage: "[Deprecated] Use: --expose-ports. - Publish a port from the main container, same format as docker --publish.", Hidden: true},
		cli.BoolFlag{Name: "attach-on-error", Usage: "Attach shell to container if a step fails.", Hidden: true},
		cli.BoolFlag{Name: "enable-volumes", Usage: "Mount local files and directories as volumes to your wercker container, specified in your wercker.yml."},
		cli.BoolFlag{Name: "enable-dev-steps", Hidden: true, Usage: `
		Enable internal dev steps.
		This enables:
		- internal/watch
		`},
	}

	// Flags for advanced deploy settings
	InternalDeployFlags = []cli.Flag{
		cli.BoolFlag{Name: "expose-ports", Usage: "Enable ports from wercker.yml beeing exposed to the host system."},
		// deprecated
		cli.StringSliceFlag{Name: "publish", Value: &cli.StringSlice{}, Usage: "[Deprecated] Use: --expose-ports. - Publish a port from the main container, same format as docker --publish.", Hidden: true},
		cli.BoolFlag{Name: "attach-on-error", Usage: "Attach shell to container if a step fails.", Hidden: true},
		cli.BoolFlag{Name: "enable-dev-steps", Hidden: true, Usage: `
		Enable internal dev steps.
		This enables:
		- internal/watch
		`},
	}

	// AWS bits
	AWSFlags = []cli.Flag{
		cli.StringFlag{Name: "aws-secret-key", Value: "", Usage: "Secret access key. Used for artifact storage."},
		cli.StringFlag{Name: "aws-access-key", Value: "", Usage: "Access key id. Used for artifact storage."},
		cli.StringFlag{Name: "s3-bucket", Value: "wercker-development", Usage: "Bucket for artifact storage."},
		cli.StringFlag{Name: "aws-region", Value: "us-east-1", Usage: "AWS region to use for artifact storage."},
	}

	// keen.io bits
	KeenFlags = []cli.Flag{
		cli.BoolFlag{Name: "keen-metrics", Usage: "Report metrics to keen.io.", Hidden: true},
		cli.StringFlag{Name: "keen-project-write-key", Value: "", Usage: "Keen write key.", Hidden: true},
		cli.StringFlag{Name: "keen-project-id", Value: "", Usage: "Keen project id.", Hidden: true},
	}

	// Wercker Reporter settings
	ReporterFlags = []cli.Flag{
		cli.BoolFlag{Name: "report", Usage: "Report logs back to wercker (requires build-id, wercker-host, wercker-token).", Hidden: true},
		cli.StringFlag{Name: "wercker-host", Usage: "Wercker host to use for wercker reporter.", Hidden: true},
		cli.StringFlag{Name: "wercker-token", Usage: "Wercker token to use for wercker reporter.", Hidden: true},
	}

	// These options might be overwritten by the wercker.yml
	ConfigFlags = []cli.Flag{
		cli.StringFlag{Name: "ignore-file", Value: ".werckerignore", Usage: "File with file patterns to ignore when copying files."},
		cli.StringFlag{Name: "source-dir", Value: "", Usage: "Source path relative to checkout root."},
		cli.Float64Flag{Name: "no-response-timeout", Value: 5, Usage: "Timeout if no script output is received in this many minutes."},
		cli.Float64Flag{Name: "command-timeout", Value: 25, Usage: "Timeout if command does not complete in this many minutes."},
		cli.StringFlag{Name: "wercker-yml", Value: "", Usage: "Specify a specific yaml file.", EnvVar: "WERCKER_YML_FILE"},
	}

	// Steps options
	StepPublishFlags = []cli.Flag{
		cli.StringFlag{Name: "owner", Value: "", Usage: "owner of the step, leave blank to use the token owner"},
	}

	PullFlagSet = [][]cli.Flag{
		[]cli.Flag{
			cli.StringFlag{Name: "branch", Value: "", Usage: "Filter on this branch."},
			cli.StringFlag{Name: "result", Value: "", Usage: "Filter on this result (passed or failed)."},
			cli.StringFlag{Name: "output", Value: "./repository.tar", Usage: "Path to repository."},
			cli.BoolFlag{Name: "load", Usage: "Load the container into docker after downloading."},
			cli.BoolFlag{Name: "f, force", Usage: "Override output if it already exists."},
		},
	}

	GlobalFlagSet = [][]cli.Flag{
		DevFlags,
		EndpointFlags,
		AuthFlags,
	}

	DockerFlagSet = [][]cli.Flag{
		DockerFlags,
	}

	PipelineFlagSet = [][]cli.Flag{
		LocalPathFlags,
		WerckerFlags,
		DockerFlags,
		InternalBuildFlags,
		GitFlags,
		RegistryFlags,
		ArtifactFlags,
		AWSFlags,
		ConfigFlags,
	}

	DeployPipelineFlagSet = [][]cli.Flag{
		LocalPathFlags,
		WerckerFlags,
		DockerFlags,
		InternalDeployFlags,
		GitFlags,
		RegistryFlags,
		ArtifactFlags,
		AWSFlags,
		ConfigFlags,
	}

	DevPipelineFlagSet = [][]cli.Flag{
		LocalPathFlags,
		WerckerFlags,
		DockerFlags,
		InternalDevFlags,
		GitFlags,
		RegistryFlags,
		ArtifactFlags,
		AWSFlags,
		ConfigFlags,
	}

	WerckerInternalFlagSet = [][]cli.Flag{
		InternalPathFlags,
		KeenFlags,
		ReporterFlags,
	}

	WerckerDockerFlagSet = [][]cli.Flag{
		AuthFlags,
		WerckerFlags,
	}
)

func FlagsFor(flagSets ...[][]cli.Flag) []cli.Flag {
	all := []cli.Flag{}
	for _, flagSet := range flagSets {
		for _, x := range flagSet {
			all = append(all, x...)
		}
	}
	return all
}
