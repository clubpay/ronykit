package preset

import (
	"math"
	"os"
	"path/filepath"

	"github.com/clubpay/ronykit/boxship"
	"github.com/clubpay/ronykit/boxship/pkg/settings"
)

func TraefikX(set settings.Settings) boxship.ContainerDesc {
	desc, err := Traefik(set)
	if err != nil {
		panic(err)
	}

	return desc
}

func Traefik(set settings.Settings) (boxship.ContainerDesc, error) {
	err := os.MkdirAll(settings.GetConfigsDir(set, settings.TraefikContainerName), os.ModePerm)
	if err != nil {
		return boxship.ContainerDesc{}, err
	}

	configDir, err := filepath.Abs(settings.GetConfigsDir(set, settings.TraefikContainerName))
	if err != nil {
		return boxship.ContainerDesc{}, err
	}

	f, err := os.Create(settings.GetConfigsDir(set, settings.Traefik, "certificates.toml"))
	if err != nil {
		return boxship.ContainerDesc{}, err
	}

	_ = f.Close()

	return boxship.ContainerDesc{
		Index:       math.MaxInt32,
		Disable:     false,
		AutoCert:    false,
		HTTPRoute:   nil,
		BuildConfig: nil,
		PullConfig: &boxship.PullConfig{
			Image: settings.TraefikImage,
		},
		Name:     settings.TraefikContainerName,
		Hostname: settings.TraefikContainerName,
		EnvFile:  "",
		Env:      nil,
		Volumes: map[string]string{
			"/var/run/docker.sock": "/var/run/docker.sock",
			"/config/":             configDir,
		},
		Labels:     nil,
		Entrypoint: nil,
		Cmd: []string{
			"--providers.docker=true",
			"--api.dashboard=true",
			"--api.insecure=true",
			"--log.level=DEBUG",
			"--entryPoints.web.address=:80",
			"--entryPoints.web.transport.respondingTimeouts.readTimeout=42",
			"--entryPoints.web.transport.respondingTimeouts.writeTimeout=42",
			"--entryPoints.web.transport.respondingTimeouts.idleTimeout=42",
			"--entryPoints.websecure.address=:443",
			"--entryPoints.websecure.transport.respondingTimeouts.readTimeout=42",
			"--entryPoints.websecure.transport.respondingTimeouts.writeTimeout=42",
			"--entryPoints.websecure.transport.respondingTimeouts.idleTimeout=42",
			"--providers.file.directory=/config/",
			"--providers.file.watch=true",
		},
		Ports: []string{"443:443", "80:80", "8080:8080"},
		Networks: []string{
			settings.TraefikNetwork,
		},
		WaitStrategy:    "",
		WaitStrategyArg: "",
		Privileged:      false,
	}, nil
}
