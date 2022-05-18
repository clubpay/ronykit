package cluster

import (
	"context"

	"github.com/clubpay/ronykit"
)

type member struct {
	ID       string   `json:"id"`
	URLs     []string `json:"urls"`
	Endpoint string   `json:"endpoint"`
}

var _ ronykit.ClusterMember = (*member)(nil)

func (m member) ServerID() string {
	return m.ID
}

func (m member) AdvertisedURL() []string {
	return m.URLs
}

func (m member) RoundTrip(ctx context.Context, sendData []byte) (receivedData []byte, err error) {
	return nil, nil
}
