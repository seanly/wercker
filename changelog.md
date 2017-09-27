## unreleased
- Update dependencies, as a result of `Sirupsen/logrus` -> `sirupsen/logrus` (#333)
- Add a Docker subcommand (#335)
- Ensure repository names are always lowercase (#338)
- Preservce timestamps when copying source and cache into container (#340)

## v1.0.965 (2017-08-23)

- Change compilation in separate wercker steps (#331)
- Add retry and exponential backoff for fetching step metadata and step tarball
  (#330)
- Add flag to delete Docker image after pushing it to a registry (#327)
- Use wercker registry for wercker-init (#334)

## v1.0.938 (2017-07-27)

- Some nice additions to the way we do the docker push and test (#320)
- Fix env var loading order (#314, #315, #317)
- Fix internal/watchstep (#312)
- Add env option to docker-scratch-push (#295)
- Allow relative paths for file:// targets in dev mode (#296)
- Better control limiting memory on run containers, when using
  services gives the services a 25% of the total memory to split
  amongst themselves, defaults to no limits (#299)
- Automatically detect bash or sh for containers by default,
  defaulting to bash if it is there (#301)
- Fix a small bug when doing local deploys and using a working-dir other
  than .wercker (#301)

## v1.0.758 (2017-01-27)

- Add Azure Registry support (#275)
- Explicitly chmods the basepath / source path to a+rx
- Removes the explicit clear after launching a shell (#257)
- Fix `wercker doc` and update `./Documentation/*` (#260)

## v1.0.643 (2016-10-05)

- Remove google as default container DNS (#245)
- Update to compiling with go 1.7

## v1.0.629 (2016-09-21)

- Add additional output when storing artifacts (#207)
- Fix longer (2+) chains of runs that have source-dir specified (#151)
- Output more descriptive error message when setup environment fails (#230)
- Allow use of an "ignore-file" yaml directive that parse the gitignore syntax
  (#240)

## v1.0.560 (2016-07-14)

- Fix internal/docker-scratch-push for Docker 1.10+

## v1.0.547 (2016-07-01)

- Add checkpointing and base-path (#123)
- Support for registry v2 (#131)
- Mount volumes in the container from different local paths (#134)
- Only push tags that were defined in the wercker.yml (#142)
- wercker is now using govendor (#146)
- Display raw config, before parsing it (#149)
- Allow multiple services with the same images (#159)
- Add exposed-ports (#161)
- Fix run, build and deploy urls (#163)

## 2016.03.11

### Features

- Moves the working path to default to `.wercker` and removes the flags
  for configuring the other paths
- Adds a symlink `.wercker/latest` for referring to your latest build, and
  a `.wercker/latest_deploy` for referring to your latest deploy
- Make the --artifacts work better locally, making your build's artifacts
  easily available under .wercker/latest/output
- Automatically use the contents of `.wercker/latest/output` when running a
  `wercker deploy` without specifying a target
- When running `wercker deploy` if the specified target does not container a
  wercker.yml file, attempt to use the one in the current directory.
- Allow settings multiple tags at a time when doing `internal/docker-push`
- Check for and allow unix:///var/run/docker.sock on non-linux hosts


### Bug Fixes

- Deal with symlinks significantly better
- Respect --docker-local when using `internal/docker-push` (don't push)
- Allow images to be pulled by nested local services (removes
  implicit --docker-local)
- Workaround a docker issue related to not fully consuming the result of a
  CopyFromContainer API call (when we exported a cache that was more than our
  limit of 1GB we'd just drop it, and docker would hang)
- Remove pipeline ID tag set by `internal/docker-push`


## 2016.02.10

### Features

- Allow users to mount local volumes to their wercker build containers, specified by a list of `volumes` underneath box in the werker.yml file. Must have `--enable-volumes` flag set in order to run.
- Check to see if config from wercker.yml is empty
- Adds changelog

### Bug fixes

- Fixes to the shellstep implementation
