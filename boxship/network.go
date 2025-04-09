package boxship

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

func BuildNetwork(ctx context.Context) *testcontainers.DockerNetwork {
	gnet, err := network.New(
		ctx,
		network.WithCheckDuplicate(),
	)
	if err != nil {
		return nil
	}

	return gnet
}
