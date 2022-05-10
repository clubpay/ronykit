package cluster

import "github.com/clubpay/ronykit"

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

func (m member) RemoteExecute(ctx *ronykit.Context) error {
	ctx.In()

	return nil
}
