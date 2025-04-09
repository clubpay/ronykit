# Getting Started

> Make sure to rename the sample.env file to .env and update it with proper values.

> You should also install the root certificate from ./setup/rootca/ca.cer file in your OS or browser.

#
# Run

run `boxship build` and `boxship run`

#
# How to Install RootCA in your OS

You need to install the root certificate located in ./setup/rootca/ca.cer into your operating
system to skip the SSL Certificate issue in browsers.

To install the root certificate, you can read the following links:

> **MacOS**: https://www.bounca.org/tutorials/install_root_certificate.html

> **Windows**: https://www.microfocus.com/documentation/visual-cobol/VC40/CSWin/HHSTSTCERT03.html

> **Firefox**: https://support.mozilla.org/en-US/kb/setting-certificate-authorities-firefox
> **Firefox**: https://wiki.mozilla.org/CA/AddRootToFirefox


# The Setup File Syntax

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
| httpRoute   | map <name, string>  | if run with --traefik flag, then it uses httpRoute data                                    |
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

| Name         | Type / SubKey | Description                                                                |
|--------------|---------------|----------------------------------------------------------------------------|
| exec         | [][]string    | script lines, each line need to be also separated like in cmd.Exec pacakge |
| runInContext | bool          | if true, the working directory will be folder of repo                      |

### HttpRouteConfig

| Name      | Type / SubKey | Description                                                            |
|-----------|---------------|------------------------------------------------------------------------|
| subdomain | string        | service will be accessible using `subdomain`.`YOUR_DOMAIN`.boxship.dev |
| tls       | bool          | if true, it will be served as https                                    |
| port      | number        | the port number that should connect in container                       |
