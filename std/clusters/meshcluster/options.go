package meshcluster

type Option func(*config)

type config struct {
	name string

	// raft
	bindAddr      string
	advertiseAddr string
	poolSize      int

	// gossip
	gossipSeeds []string
}
