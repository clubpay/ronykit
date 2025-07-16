package boxship

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/clubpay/ronykit/boxship/pkg/log"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/clubpay/ronykit/boxship/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/joho/godotenv"
	"github.com/moby/term"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/fx"
)

const (
	ctxTimeout  = time.Second * 10
	logInterval = time.Second * 3
)

type Config struct {
	fx.In

	Logger   log.Logger
	Settings settings.Settings
}

type Context struct {
	dockerC *client.Client
	log     log.Logger
	set     settings.Settings

	containers     map[string]ContainerDesc
	containersSort []ContainerDesc
	containersRun  map[string]*Container
}

func NewContext(cfg Config) (*Context, error) {
	dockerC, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	_, err = dockerC.Ping(context.TODO())
	if err != nil {
		return nil, err
	}

	b := &Context{
		dockerC:       dockerC,
		log:           cfg.Logger,
		set:           cfg.Settings,
		containers:    map[string]ContainerDesc{},
		containersRun: map[string]*Container{},
	}

	err = b.prepare()

	return b, err
}

func (buildCtx *Context) prepare() error {
	perm := os.ModePerm
	_ = os.MkdirAll(buildCtx.LogsDir(), perm)
	_ = os.MkdirAll(buildCtx.RepoDir(), perm)
	_ = os.MkdirAll(buildCtx.ConfigsDir(), perm)
	_ = os.MkdirAll(buildCtx.CertsDir(), perm)
	_ = os.MkdirAll(buildCtx.VolumeDir(), perm)

	index := 1
	buildCtx.containersSort = buildCtx.containersSort[:0]

	return readYamlDir(
		buildCtx.Context(), buildCtx.set.GetString(settings.Setup),
		func(ctx context.Context, yamlFilePath string) error {
			sf, err := parseYamlFile(yamlFilePath, buildCtx.set.GetString)
			if err != nil {
				return err
			}

			for _, desc := range sf.Containers {
				if desc.Index == 0 {
					desc.Index = index
				}

				desc.Networks = sf.Networks
				desc.SetDefaultGitAuth(sf.DefaultGitAuth)
				desc.SetDefaultRegistryCred(sf.DefaultRegistryCred)

				buildCtx.RegisterContainerDesc(desc)

				index++
			}

			sort.SliceStable(buildCtx.containersSort, func(i, j int) bool {
				return buildCtx.containersSort[i].Index < buildCtx.containersSort[j].Index
			})

			return nil
		},
	)
}

func (buildCtx *Context) getContainerDesc(name string) (*ContainerDesc, error) {
	desc, ok := buildCtx.containers[name]
	if !ok {
		return nil, fmt.Errorf("container description not found: %s", name)
	}

	return &desc, nil
}

func (buildCtx *Context) CreateNetwork(name string) error {
	res, err := buildCtx.dockerC.NetworkCreate(buildCtx.Context(), name, network.CreateOptions{})
	if err != nil {
		return err
	}

	buildCtx.Log().Infof("Network created: %s, %s", res.ID, res.Warning)

	return nil
}

func (buildCtx *Context) RegisterContainerDesc(desc ContainerDesc) {
	if _, err := buildCtx.getContainerDesc(desc.Name); err != nil {
		buildCtx.containers[desc.Name] = desc
		buildCtx.containersSort = append(buildCtx.containersSort, desc)
	}
}

func (buildCtx *Context) Consume(c *Container) {
	rd, err := buildCtx.dockerC.ContainerLogs(
		context.Background(), c.GetContainerID(),
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    true,
		},
	)
	if err != nil {
		buildCtx.log.Warnf("[%s] logger read error: (%v)", c.Name, err)

		return
	}

	go func(c *Container) {
		for {
			_, _ = stdcopy.StdCopy(c.Logger, c.Logger, rd)

			ctx, cf := context.WithTimeout(context.Background(), ctxTimeout)
			state, err := c.tc.State(ctx)

			cf()

			if err != nil || !state.Running {
				buildCtx.log.Debugf("[%s] logger closed: (%v)", c.Name, err)

				break
			}

			time.Sleep(logInterval)
		}

		_ = rd.Close()
		_ = c.Logger.Close()
	}(c)
}

func (buildCtx *Context) Context() context.Context {
	return context.Background()
}

func (buildCtx *Context) Log() log.Logger {
	return buildCtx.log
}

func (buildCtx *Context) Settings() settings.Settings {
	return buildCtx.set
}

func (buildCtx *Context) CertsDir(elems ...string) string {
	return settings.GetCertsDir(buildCtx.set, elems...)
}

func (buildCtx *Context) ConfigsDir(elems ...string) string {
	return settings.GetConfigsDir(buildCtx.set, elems...)
}

func (buildCtx *Context) LogsDir(elems ...string) string {
	return settings.GetLogsDir(buildCtx.set, elems...)
}

func (buildCtx *Context) RepoDir(elems ...string) string {
	return settings.GetRepoDir(buildCtx.set, elems...)
}

func (buildCtx *Context) VolumeDir(elems ...string) string {
	return settings.GetVolumesDir(buildCtx.set, elems...)
}

func (buildCtx *Context) Domain() string {
	return buildCtx.set.GetString(settings.Domain)
}

func (buildCtx *Context) ForEachContainer(f func(desc ContainerDesc)) {
	for _, desc := range buildCtx.containersSort {
		f(desc)
	}
}

func (buildCtx *Context) BuildImage(name string) error {
	desc, err := buildCtx.getContainerDesc(name)
	if err != nil {
		return err
	}

	if !desc.ShouldBuild() {
		return nil
	}

	if desc.BuildConfig.Git != nil {
		buildCtx.log.Debugf("[%s] cloning git repo", desc.Name)
		err := buildCtx.cloneRepo(desc)
		if err != nil {
			return err
		}

		buildCtx.log.Debugf("[%s] git repo cloned", desc.Name)
	}

	if desc.BuildConfig.BeforeBuild != nil {
		buildCtx.log.Debugf("[%s] executing pre-built actions", desc.Name)
		err := runAction(buildCtx.RepoDir(desc.Name), desc.BuildConfig.BeforeBuild.Exec...)
		if err != nil {
			return err
		}
	}

	dockerBuildCtx, err := archive.TarWithOptions(desc.BuildConfig.Src.Context, &archive.TarOptions{})
	if err != nil {
		return err
	}

	buildOptions := types.ImageBuildOptions{
		BuildArgs:   desc.GetBuildArgs(),
		Dockerfile:  desc.GetDockerfile(),
		Context:     dockerBuildCtx,
		Tags:        []string{fmt.Sprintf("%s:boxship", desc.Name)},
		Remove:      true,
		ForceRemove: true,
		CacheFrom:   []string{fmt.Sprintf("%s:boxship", desc.Name)},
	}
	if buildCtx.Settings().GetBool(settings.BuildKit) {
		buildOptions.Version = types.BuilderBuildKit
	}

	buildCtx.Log().Infof("[%s] building docker image", desc.Name)

	resp, err := buildCtx.dockerC.ImageBuild(buildCtx.Context(), dockerBuildCtx, buildOptions)
	if err != nil {
		return err
	}

	buildCtx.Log().Infof("[%s] docker image build request sent", desc.Name)

	termFd, isTerm := term.GetFdInfo(
		buildCtx.log.FileLogger(filepath.Join(buildCtx.LogsDir(), fmt.Sprintf("%s-build.log", desc.Name))),
	)

	err = jsonmessage.DisplayJSONMessagesStream(resp.Body, buildCtx.log, termFd, isTerm, nil)
	if err != nil {
		return err
	}

	_, err = io.Copy(buildCtx.log, resp.Body)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	if desc.BuildConfig.AfterBuild != nil {
		buildCtx.log.Debugf("[%s] executing after-built actions", desc.Name)

		err = runAction(buildCtx.RepoDir(desc.Name), desc.BuildConfig.AfterBuild.Exec...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (buildCtx *Context) BuildNetwork(name string) error {
	desc, err := buildCtx.getContainerDesc(name)
	if err != nil {
		return err
	}

	netSummary, err := buildCtx.dockerC.NetworkList(
		buildCtx.Context(),
		network.ListOptions{},
	)
	if err != nil {
		return err
	}

	for _, net := range desc.Networks {
		if slices.ContainsFunc(
			netSummary,
			func(summary network.Summary) bool { return summary.Name == net },
		) {
			continue
		}

		_, err = buildCtx.dockerC.NetworkCreate(
			buildCtx.Context(), net, network.CreateOptions{
				Driver: "bridge",
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (buildCtx *Context) cloneRepo(desc *ContainerDesc) error {
	if buildCtx.Settings().GetBool(settings.ShallowClone) {
		return buildCtx.shallowClone(desc)
	} else {
		wd := buildCtx.RepoDir(desc.Name)

		err := buildCtx.deepClone(desc)
		if err != nil {
			// if got error, then we try one more time by deleting the repo folder. Usually that fixes
			// many problems :)
			err = os.RemoveAll(wd)
			if err != nil {
				buildCtx.Log().Warnf("[%s] got error on deleting repo folder", desc.Name)

				return err
			}

			return buildCtx.deepClone(desc)
		}
	}

	return nil
}

func (buildCtx *Context) deepClone(desc *ContainerDesc) error {
	if desc.BuildConfig.Git.Remote == "" {
		desc.BuildConfig.Git.Remote = "origin"
	}

	wd := buildCtx.RepoDir(desc.Name)
	desc.BuildConfig.Src = &LocalSource{
		Context: wd,
	}

	if desc.BuildConfig.Git.Branch == "" {
		desc.BuildConfig.Git.Branch = "master"
	}

	_, err := git.PlainClone(
		wd, false,
		&git.CloneOptions{
			URL:           desc.BuildConfig.Git.Repo,
			ReferenceName: plumbing.NewBranchReferenceName(desc.BuildConfig.Git.Branch),
			Auth: &http.BasicAuth{
				Username: desc.BuildConfig.Git.User,
				Password: desc.BuildConfig.Git.Pass,
			},
			Progress:     buildCtx.log,
			SingleBranch: true,
		},
	)
	if err == nil {
		return nil
	}

	if !errors.Is(err, git.ErrRepositoryAlreadyExists) {
		return err
	}

	r, err := git.PlainOpenWithOptions(wd, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return err
	}

	wt, err := r.Worktree()
	if err != nil {
		return err
	}

	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(desc.BuildConfig.Git.Branch),
		Create: false,
		Force:  true,
		Keep:   false,
	})
	if err != nil {
		buildCtx.Log().Infof("[%s] could not checkout: %v", desc.Name, err)
	}

	headRef, err := r.Head()
	if err != nil {
		return err
	}

	headCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}

	buildCtx.Log().Infof("[%s] git-repo cloned with git-hash: %s", desc.Name, headCommit.Hash.String())

	err = wt.Pull(&git.PullOptions{
		Auth: &http.BasicAuth{
			Username: desc.BuildConfig.Git.User,
			Password: desc.BuildConfig.Git.Pass,
		},
		Progress:      buildCtx.Log(),
		ReferenceName: plumbing.NewBranchReferenceName(desc.BuildConfig.Git.Branch),
		RemoteName:    desc.BuildConfig.Git.Remote,
		Force:         true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	return nil
}

func (buildCtx *Context) shallowClone(desc *ContainerDesc) error {
	if desc.BuildConfig.Git.Remote == "" {
		desc.BuildConfig.Git.Remote = "origin"
	}

	wd := buildCtx.RepoDir(desc.Name)
	desc.BuildConfig.Src = &LocalSource{
		Context: wd,
	}

	if desc.BuildConfig.Git.Branch == "" {
		desc.BuildConfig.Git.Branch = "master"
	}

	err := os.RemoveAll(wd)
	if err != nil {
		buildCtx.Log().Warnf("[%s] got error on deleting repo folder", desc.Name)

		return err
	}

	_, err = git.PlainClone(
		wd, false,
		&git.CloneOptions{
			URL:           desc.BuildConfig.Git.Repo,
			ReferenceName: plumbing.NewBranchReferenceName(desc.BuildConfig.Git.Branch),
			Auth: &http.BasicAuth{
				Username: desc.BuildConfig.Git.User,
				Password: desc.BuildConfig.Git.Pass,
			},
			Progress:     buildCtx.Log(),
			SingleBranch: true,
			Depth:        1,
		},
	)

	return nil
}

func (buildCtx *Context) PullImage(name string) error {
	desc, err := buildCtx.getContainerDesc(name)
	if err != nil {
		return err
	}

	buildCtx.log.Debugf("[%s] pulling docker image", desc.Name)

	resp, err := buildCtx.dockerC.ImagePull(buildCtx.Context(), desc.PullConfig.Image, image.PullOptions{
		All:           false,
		RegistryAuth:  desc.GetRegistryAuth(),
		PrivilegeFunc: nil,
		Platform:      "",
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(buildCtx.log, resp)

	err = resp.Close()
	if err != nil {
		return err
	}

	buildCtx.log.Debugf("[%s] image pulled", desc.Name)

	return nil
}

// RunContainer runs the container and returns its id.
func (buildCtx *Context) RunContainer(name string) (string, error) {
	desc, err := buildCtx.getContainerDesc(name)
	if err != nil {
		return "", err
	}

	c, err := buildCtx.buildContainer(desc)
	if err != nil {
		return "", err
	}

	err = c.Run(buildCtx)
	if err != nil {
		return "", err
	}

	buildCtx.containersRun[c.GetContainerID()] = c

	return c.GetContainerID(), nil
}

func (buildCtx *Context) buildContainer(desc *ContainerDesc) (*Container, error) {
	buildCtx.log.Infof("[%s] building container", desc.Name)

	if desc.Hostname == "" {
		desc.Hostname = desc.Name
	}

	if desc.HTTPRoute != nil {
		err := buildCtx.setTraefik(desc)
		if err != nil {
			return nil, err
		}
	}

	if desc.AutoCert {
		err := buildCtx.genAutoCert(desc)
		if err != nil {
			return nil, err
		}
	}

	req := testcontainers.ContainerRequest{
		Name:         desc.Name,
		Hostname:     desc.Hostname,
		Image:        desc.GetImage(),
		Env:          desc.Env,
		Labels:       desc.Labels,
		Cmd:          desc.Cmd,
		Entrypoint:   desc.Entrypoint,
		ExposedPorts: desc.Ports,
		Networks:     desc.Networks,
		Privileged:   desc.Privileged,
	}

	for k, v := range desc.Volumes {
		targetMount, err := filepath.Abs(k)
		if err != nil {
			return nil, err
		}

		if v == "dynamic" {
			v = buildCtx.VolumeDir(desc.Name)

			err = os.MkdirAll(v, os.ModeDir|os.ModePerm)
			if err != nil {
				return nil, err
			}
		}

		hostPath, err := filepath.Abs(v)
		if err != nil {
			return nil, err
		}

		req.Mounts = append(
			req.Mounts,
			testcontainers.BindMount(hostPath, testcontainers.ContainerMountTarget(targetMount)),
		)
	}

	req.NetworkAliases = map[string][]string{}
	for _, net := range desc.Networks {
		req.NetworkAliases[net] = append(desc.Alias, desc.Name)
	}

	switch desc.WaitStrategy {
	case WaitForExit:
		req.WaitingFor = wait.ForExit()
	case WaitForLog:
		req.WaitingFor = wait.ForLog(desc.WaitStrategyArg)
	case WaitForHttp:
		req.WaitingFor = wait.ForHTTP(desc.WaitStrategyArg)
	case WaitForHealthCheck:
		req.WaitingFor = wait.ForHealthCheck()
	case WaitForListeningPort:
		req.WaitingFor = wait.ForListeningPort(nat.Port(desc.WaitStrategyArg)).WithStartupTimeout(time.Second * 30)
	case WaitForExec:
		req.WaitingFor = wait.ForExec(strings.Split(desc.WaitStrategyArg, " "))
	case WaitForSQL:
		req.WaitingFor = wait.ForSQL(
			"80", "postgres",
			func(host string, port nat.Port) string {
				return desc.WaitStrategyArg
			},
		)
	}

	if desc.EnvFile != "" {
		err := buildCtx.loadEnv(desc, &req)
		if err != nil {
			return nil, err
		}
	}

	tc, err := testcontainers.GenericContainer(
		buildCtx.Context(),
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
		},
	)
	if err != nil {
		return nil, err
	}

	name, err := tc.Name(buildCtx.Context())
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(buildCtx.LogsDir(), os.ModeDir|os.ModePerm)
	if err != nil {
		return nil, err
	}

	rtLogger, err := os.Create(filepath.Join(buildCtx.LogsDir(), fmt.Sprintf("%s.log", name)))
	if err != nil {
		return nil, err
	}

	c := &Container{
		desc:   desc,
		tc:     tc,
		Name:   name,
		ID:     tc.GetContainerID(),
		Logger: rtLogger,
	}

	buildCtx.log.Infof("[%s] container created: %s", desc.Name, c.GetContainerID())

	return c, nil
}

func (buildCtx *Context) loadEnv(desc *ContainerDesc, req *testcontainers.ContainerRequest) error {
	if req.Env == nil {
		req.Env = map[string]string{}
	}

	envMap, err := godotenv.Read(desc.EnvFile)
	if err != nil {
		return err
	}

	for k, v := range envMap {
		foundDynamicVars := regexEnvFileKeys.FindString(v)
		if foundDynamicVars != "" {
			x := strings.Trim(foundDynamicVars, "{}")

			foundX := buildCtx.set.GetString(x)
			if foundX != "" {
				v = strings.ReplaceAll(v, foundDynamicVars, foundX)
			}
		}

		req.Env[k] = v
	}

	return nil
}

func (buildCtx *Context) setTraefik(desc *ContainerDesc) error {
	entrypoint := settings.TraefikEndpoint

	if desc.Labels == nil {
		desc.Labels = map[string]string{}
	}

	entryPointKey := fmt.Sprintf("traefik.http.routers.%s.entrypoints", desc.Name)
	ruleKey := fmt.Sprintf("traefik.http.routers.%s.rule", desc.Name)
	tlsKey := fmt.Sprintf("traefik.http.routers.%s.tls", desc.Name)
	lbPortKey := fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", desc.Name)
	lbScheme := fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.scheme", desc.Name)
	host := fmt.Sprintf("%s.%s", desc.HTTPRoute.SubDomain, buildCtx.Domain())

	// buildCtx.Log().Debugf("[%s] expose http route through traefik: %s", desc.Name, host)

	if desc.HTTPRoute.TLS {
		entrypoint = settings.TraefikSecureEndpoint
		desc.Labels[tlsKey] = "true"
	}

	desc.Labels["traefik.enable"] = "true"
	desc.Labels["traefik.docker.network"] = settings.TraefikNetwork
	desc.Labels[entryPointKey] = entrypoint
	desc.Labels[ruleKey] = fmt.Sprintf("Host(`%s`)", host)
	desc.Labels[lbPortKey] = desc.HTTPRoute.Port
	desc.Labels[lbScheme] = "http"
	desc.Networks = append(desc.Networks, settings.TraefikNetwork)

	rootCATLS, err := utils.GetCertificate(
		buildCtx.set.GetString(settings.CAKeyFile),
		buildCtx.set.GetString(settings.CACertFile),
	)
	if err != nil {
		return err
	}

	rootCA, err := x509.ParseCertificate(rootCATLS.Certificate[0])
	if err != nil {
		return err
	}

	certTmpl := utils.CertTemplate("TR", "Boxship", host)

	_ = os.MkdirAll(buildCtx.ConfigsDir(settings.Traefik, "certs"), os.ModePerm)
	utils.GenerateCert(certTmpl, rootCA,
		buildCtx.Settings().GetString(settings.CAKeyFile),
		buildCtx.ConfigsDir(settings.Traefik, "certs", fmt.Sprintf("%s.key.pem", host)),
		buildCtx.ConfigsDir(settings.Traefik, "certs", fmt.Sprintf("%s.cert.pem", host)),
	)

	f, err := os.OpenFile(
		buildCtx.ConfigsDir(settings.Traefik, "certificates.toml"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm, //nolint:nosnakecase
	)
	if err != nil {
		return err
	}

	_, _ = f.WriteString("[[tls.certificates]]\n")
	_, _ = f.WriteString(fmt.Sprintf("  keyFile = \"/config/certs/%s.key.pem\"\n", host))
	_, _ = f.WriteString(fmt.Sprintf("  certFile = \"/config/certs/%s.cert.pem\"\n\n", host))

	return f.Close()
}

func (buildCtx *Context) genAutoCert(desc *ContainerDesc) error {
	rootCATLS, err := utils.GetCertificate(
		buildCtx.set.GetString(settings.CAKeyFile),
		buildCtx.set.GetString(settings.CACertFile),
	)
	if err != nil {
		return err
	}

	rootCA, err := x509.ParseCertificate(rootCATLS.Certificate[0])
	if err != nil {
		return err
	}

	certTmpl := utils.CertTemplate("TR", desc.Hostname, desc.Hostname, desc.AutoCertDNS...)
	_ = os.MkdirAll(buildCtx.CertsDir(desc.Hostname), os.ModePerm)
	utils.GenerateCert(certTmpl, rootCA,
		buildCtx.set.GetString(settings.CAKeyFile),
		buildCtx.CertsDir(desc.Hostname, "key.pem"),
		buildCtx.CertsDir(desc.Hostname, "cert.pem"),
	)

	if desc.Volumes == nil {
		desc.Volumes = map[string]string{}
	}

	buildCtx.log.Debugf("[%s] generate certificate: %s",
		desc.Name, desc.Hostname,
	)
	buildCtx.log.Debugf("[%s] mount root-ca: %s", desc.Name, buildCtx.Settings().GetString(settings.CACertFile))

	desc.Volumes["/boxship/certs"] = buildCtx.CertsDir(desc.Name)
	desc.Volumes["/boxship/ca.crt"] = buildCtx.Settings().GetString(settings.CACertFile)

	return nil
}

func (buildCtx *Context) StopContainer(id string) error {
	c := buildCtx.containersRun[id]
	if c != nil {
		return c.Terminate(buildCtx)
	}

	return nil
}

func runAction(dir string, actions ...[]string) error {
	for _, cmdArgs := range actions {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Env = os.Environ()
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
