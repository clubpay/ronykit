package settings

// Traefik settings
const (
	TraefikNetwork        = "_traefik_network"
	TraefikSecureEndpoint = "websecure"
	TraefikEndpoint       = "web"
	TraefikContainerName  = "traefik"
	TraefikImage          = "traefik:3.3.4"
)

// Settings Keys
const (
	WorkDir      = "work.dir"
	LogAll       = "log.all"
	Setup        = "setup"
	Template     = "template"
	Traefik      = "traefik"
	Domain       = "domain"
	OutputDir    = "out"
	Force        = "force"
	ShallowClone = "shallow.clone"
	BuildKit     = "build-kit"
	YamlFile     = "yaml"

	CACertFile = "ca.cert.file"
	CAKeyFile  = "ca.key.file"

	BuildContainerTimeout = "build.container.timeout"
	BuildNetworkTimeout   = "build.network.timeout"
)
