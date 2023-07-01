package meshcluster

type Option func(*config)

type config struct {
	name   string
	dbPath string

	// raft
	bindAddr      string
	advertiseAddr string
	poolSize      int

	// gossip
	gossipSeeds []string
}

func WithDBPath(path string) Option {
	return func(cfg *config) {
		cfg.dbPath = path
	}
}
