package boxship

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/clubpay/ronykit/boxship/pkg/utils"
)

// BuildConfig holds information that Boxship could build the image based on the source.
// Source is either GitSource or LocalSource
// Args is the dictionary that Boxship will pass as docker-build arguments.
// Logs if set, then Boxship stores the build logs in the log directory.
type BuildConfig struct {
	Git         *GitSource        `yaml:"git,omitempty"`
	Src         *LocalSource      `yaml:"src,omitempty"`
	DockerFile  string            `yaml:"dockerFile"`
	Args        map[string]string `yaml:"args"`
	Logs        bool              `yaml:"logs"`
	BeforeBuild *BuildAction      `yaml:"beforeBuild"`
	AfterBuild  *BuildAction      `yaml:"afterBuild"`
}

// GitSource holds information about the remote repository. Boxship uses this information
// to clone the repo and then tries to build the docker container.
type GitSource struct {
	Repo   string `yaml:"repo"`
	Remote string `yaml:"remote"`
	Branch string `yaml:"branch"`
	User   string `yaml:"user"`
	Pass   string `yaml:"pass"`
}

// LocalSource holds information about a context folder. Boxship uses this source
// folder defined with Context field to build the docker.
type LocalSource struct {
	Context string `yaml:"context"`
	Watch   bool   `yaml:"watch"`
}

// PullConfig holds information on how to pull a docker image.
type PullConfig struct {
	Image           string            `yaml:"image"`
	AlwaysPullImage bool              `yaml:"alwaysPullImage"`
	RegistryCred    map[string]string `yaml:"registryCred"`
}

// HTTPRouteConfig holds information about how the container is going to be handled
// by Traefik reverse proxy.
type HTTPRouteConfig struct {
	SubDomain string `yaml:"subdomain"`
	TLS       bool   `yaml:"tls"`
	Port      string `yaml:"port"`
}

type BuildAction struct {
	Exec [][]string `yaml:"exec"`
	// WorkingDir is the path that execution is going to happen.
	// If RunMode is 'container' this MUST be an absolute path starting with `/`
	//  If RunMode is 'host' path could be relative.
	// Default is the src dir.
	WorkingDir string `yaml:"workingDir"`
}

type RunAction struct {
	Exec [][]string `yaml:"exec"`
	// WorkingDir is the path that execution is going to happen.
	// If RunMode is 'container' this MUST be an absolute path starting with `/`
	//  If RunMode is 'host' path could be relative.
	// Default is the src dir.
	WorkingDir string        `yaml:"workingDir"`
	RunMode    ActionRunMode `yaml:"runMode"`
}

type ActionRunMode string

const (
	ActionRunModeExec      ActionRunMode = "host"
	ActionRunModeContainer ActionRunMode = "container"
)

type Enhancements struct {
	GoBuild bool `yaml:"improveGoBuild"`
	NPM     bool `yaml:"improveNPM"`
}

type ContainerDesc struct {
	// Index is the order of the container that will be started if there are multiple containers in the
	// run command, it Boxship starts to execute them in the order of the index.
	Index int `yaml:"index"`
	// Disable is the flag that Boxship uses to ignore the container in the run or build commands.
	Disable bool `yaml:"disable"`
	// AutoCert is the flag that Boxship uses to enable automatic SSL certificate generation for the container.
	// If AutoCert is set to true, then Boxship generate ssl certificates for all the domains listed in AutoCertDNS.
	AutoCert    bool     `yaml:"autoCert"`
	AutoCertDNS []string `yaml:"autoCertDNS"`
	// HTTPRoute is the configuration that Boxship uses to handle the container by Traefik reverse proxy.
	// This is only used if Traefik is enabled.
	HTTPRoute *HTTPRouteConfig `yaml:"httpRoute"`
	// BuildConfig contains configuration for `build` command if we want to build the image from the source or
	// a git repository
	BuildConfig *BuildConfig `yaml:"build,omitempty"`
	// PullConfig contains configuration for the `build` command if we want to pull the image
	PullConfig      *PullConfig       `yaml:"pull,omitempty"`
	Name            string            `yaml:"name"`
	Hostname        string            `yaml:"hostname"`
	EnvFile         string            `yaml:"envFile,omitempty"`
	Env             map[string]string `yaml:"env"`
	Volumes         map[string]string `yaml:"volumes"`
	Labels          map[string]string `yaml:"labels"`
	Entrypoint      []string          `yaml:"entrypoint"`
	Cmd             []string          `yaml:"cmd"`
	Ports           []string          `yaml:"ports"`
	Networks        []string          `yaml:"networks"`
	Alias           []string          `yaml:"alias"`
	WaitStrategy    WaitStrategy      `yaml:"waitStrategy"`
	WaitStrategyArg string            `yaml:"waitStrategyArg"`
	Privileged      bool              `yaml:"privileged"`
	AutoExec        *RunAction        `yaml:"autoExec"`
	Enhancements    Enhancements      `yaml:"enhancements"`
}

// GetBuildArgs returns the env args to be used when creating from Dockerfile
func (desc *ContainerDesc) GetBuildArgs() map[string]*string {
	args := map[string]*string{}
	for k, v := range desc.BuildConfig.Args {
		args[k] = utils.StringPtr(v)
	}

	args["TARGETARCH"] = utils.StringPtr(runtime.GOARCH)
	args["BUILDKIT_INLINE_CACHE"] = utils.StringPtr("true")

	return args
}

// GetDockerfile returns the Dockerfile from the ContainerRequest, defaults to "Dockerfile"
func (desc *ContainerDesc) GetDockerfile() string {
	if desc.BuildConfig != nil {
		return desc.BuildConfig.DockerFile
	}

	return "Dockerfile"
}

func (desc *ContainerDesc) ShouldBuild() bool {
	return desc.BuildConfig != nil && (desc.BuildConfig.Git != nil || desc.BuildConfig.Src != nil)
}

func (desc *ContainerDesc) Watched() bool {
	if desc.BuildConfig == nil || desc.BuildConfig.Src == nil {
		return false
	}

	return desc.BuildConfig.Src.Watch
}

func (desc *ContainerDesc) GetRegistryAuth() string {
	registryCredBytes, err := json.Marshal(desc.PullConfig.RegistryCred)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(registryCredBytes)
}

func (desc *ContainerDesc) GetImage() string {
	switch {
	case desc.PullConfig != nil:
		return desc.PullConfig.Image
	case desc.BuildConfig != nil:
		return fmt.Sprintf("%s:boxship", desc.Name)
	}

	panic("either pull or build config must be set")
}

func (desc *ContainerDesc) SetDefaultGitAuth(ga GitAuth) {
	if desc.BuildConfig == nil || desc.BuildConfig.Git == nil {
		return
	}

	if desc.BuildConfig.Git.User == "" && desc.BuildConfig.Git.Pass == "" {
		desc.BuildConfig.Git.User = ga.User
		desc.BuildConfig.Git.Pass = ga.Pass
	}
}

func (desc *ContainerDesc) SetDefaultRegistryCred(ra RegistryCred) {
	if desc.PullConfig == nil {
		return
	}

	if desc.PullConfig.RegistryCred == nil {
		desc.PullConfig.RegistryCred = ra
	}
}
