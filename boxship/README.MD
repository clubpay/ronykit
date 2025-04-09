![alt text](boxship.jpg?raw=true)

# BoxShip

A development environment prepare the cluster in you local computer or in CI/CD container.
With BoxShip you will be able to build containers by pulling images from registry, or building from
the Dockerfile.

# Index

1. [Getting Started](#getting-started)
2. [Simple and Powerful DSL](#simple-and-powerful-dsl)
    1. [Flag](#flags)
    2. [SetupFile Reference](#setupfile-reference)
3. [How To Install Root Certificate](#how-to-install-rootca-in-your-os)
4. [Features](#features)

---

# Getting Started

## Build / Install (Requires Golang 1.23+)

to make sure you have correct golang installed in your system, enter: `go version` it must be
greater than 1.23.0.
To build and install the executable in your machine from source code run `go install github.com/clubpay/ronykit/boxship/cmd/boxship` in
the root of the project.

## Download

If you don't have golang in your system, and you don't want to install golang in your system
you can download pre-built packages from this repo for your os.
Download the package based on your os/machine and copy the `boxship` executable into your PATH,
then run `boxship init` in a folder that you want to set up your development environment.

## Run

You need to initialize your development environment by calling `boxship init <target>`. Then you *MUST*
change your working directory to the new directory that was created by boxship init. `cd <target>`.
Now if you run `ls` or `dir` command you see there is a setup folder and a `sample.env` file.
You *MUST* rename it to `.env` file and fill it with your values. The file itself has comments to help
you through the process.

When you finished the preparation, which hopefully you need to do it only once. Then you can run
`boxship run` to start your environment. It might takes some time to bring up if you run it first time
or if any of the components have been updated, since boxship will re-create the docker image if it
finds new changes in each repo.

## Update

From version v0.2.2 boxship could update itself, just you need to run `boxship update` then it will update itself
to the latest version. Just make sure you have set GITHUB_PAT in your .env file, since boxship repo is a private
repository.

## UpdateTemplate

In some cases you want to change the setup folder yaml to some other configuration, just run `boxship update-template`
this command re-creates the `setup` folder configuration files.

[GoTop ^](#index)

# How to Install RootCA in your OS

To install root certificate you can read following links:
> MacOS: https://www.bounca.org/tutorials/install_root_certificate.html

> Windows: https://www.microfocus.com/documentation/visual-cobol/VC40/CSWin/HHSTSTCERT03.html

> https://support.mozilla.org/en-US/kb/setting-certificate-authorities-firefox

You need to install the root certificate located in ./setup/rootca/ca.cer into your operating
system.

---

[GoTop ^](#index)

# Simple and Powerful DSL

## Flags

|   Flag   |                                Description                                 |
|:--------:|:--------------------------------------------------------------------------:|
|  setup   | the directory where that yaml files are there. defaults to ./setup folder. |
|          |        if you don't provide it then we use the default setup files.        |
| log.all  |           start recording logs in the `work.dir/logs` directory.           |
| work.dir |       the directory to save temporary state files. default is ./_hdd       |

## SetupFile Reference

### Top Level Keys

| Name       | Type                  | Description                                    |
|------------|-----------------------|:-----------------------------------------------|
| networks   | array <string>        | name of networks to create                     |
| containers | map <name, container> | each key holds information about one container |

### Container

| Name        | Type                | Description                                                                                |
|-------------|---------------------|--------------------------------------------------------------------------------------------|
| disable     | bool                | if set to true, builder skip building this. this is good to temporary disable a container. |
| depends     | array <string>      | list of container name (key) that this container depends on                                |
| name        | string              | the name of the container which is shown in `docker ps` command `name` column.             |
| alias       | array <string>      | the alias names of the container which other containers in network can see                 |
| env         | map<string><string> | the environment variables available to the container in runtime                            |
| labels      | map<string><string> | the labels of the container                                                                |
| volumes     | map<string><string> | key is the path inside the container, value is the path in host machine                    |
| cmd         | array <string>      | commands overrides the actual CMD in the docker image.                                     |
| entrypoint  | string              | entrypoint overrides the actual ENTRYPOINT of the docker image                             |
| ports       | array <string>      | exposed ports                                                                              |
| privileged  | bool                | run in privileged mode                                                                     |
| httpRoute   | httpRouteConfig     | if run with --traefik flag, then it uses httpRoute data                                    |
| autoCert    | bool                | boxship generate certificates automatically based on the hostname, name and autoCertDNS    |
| autoCertDNS | array <string>      | list of domain names that certificate should be generated when autoCert is true            |
| build       | buildConfig         | if you need to build the container, check Build table for details                          |
| pull        | object              | if you need to pull the container from a docker registry, check Pull table for details     |
| beforeBuild | actionConfig        | these commands run before build process starts.                                            |
| afterBuild  | actionConfig        | these command run after build process finished.                                            |
| beforeStart | actionConfig        | these commands run before running the container.                                           |
| afterStart  | actionConfig        | these commands run after running the container.                                            |

### BuildConfig

| Name       | Type / SubKey       | Description                                                               |
|------------|---------------------|---------------------------------------------------------------------------|
| dockerFile | string              | the name of the Dockerfile to build. default is Dockerfile                |
| logs       | bool                | to save the build logs and save them in Working directory logs sub-folder |
| args       | map<string><string> | the build arguments for build docker                                      |
| src        |                     | if we want to build from source                                           |
|            | context             | the path of the project which holds the docker file                       |
| git        |                     | if we want to pull the source and then build                              |
|            | repo                | the path of the repo. e.g. https://github.com/clubpay/boxship           |
|            | branch              | the branch. default is master                                             |
|            | execCmd             | this is an array of array of strings. These are commands run in           |
|            |                     | local machine in the root path of the repo folder. It is useful           |
|            |                     | when you need to do some preparation before building the Dockerfile       |

### ActionConfig

| Name       | Type / SubKey | Description                                                                  |
|------------|---------------|------------------------------------------------------------------------------|
| exec       | [][]string    | script lines, each line need to be also separated like in cmd.Exec pacakge   |
| workingDir | string        | if runMode is `container` it MUST be absolute.                               |
| runMode    | enum          | possible values: container, host. `container` is ONLY valid for `afterStart` |

### HttpRouteConfig

| Name      | Type / SubKey | Description                                                            |
|-----------|---------------|------------------------------------------------------------------------|
| subdomain | string        | service will be accessible using `subdomain`.`YOUR_DOMAIN`.boxship.dev |
| tls       | bool          | if true, it will be served as https                                    |
| port      | number        | the port number that should connect in container                       |

---

[GoTop ^](#index)

# Features

- [x] Build From YML setup files
- [x] Build From YML setup directory
- [x] Pull Images
- [x] Pull GIT repo
- [x] Build from Dockerfile
- [ ] Define and Check Dependency Graph
- [x] Saving container logs in separate files.
- [x] Saving container build logs in separate files
- [x] Record Container Logs
- [x] Support Dynamic Variables
- [ ] Auto-build by watching folders
- [x] Support Action Scripts
- [ ] Interactive Shell
    - [ ] Logs
    - [ ] Stop Container
    - [ ] Start Container
    - [ ] Restart Container
    - [ ] Scale Up/Down Container
