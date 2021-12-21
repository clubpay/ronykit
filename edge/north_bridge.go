package edge

import (
	"github.com/ronaksoft/ronykit"
)

type northBridge struct {
	gw ronykit.Gateway
	d  ronykit.Dispatcher
}

func (n northBridge) OnOpen(c ronykit.GatewayConn) {
	n.d.OnOpen(c)
}

func (n northBridge) OnClose(connID uint64) {
	// TODO:: do we need to do anything ?
}

func (n northBridge) OnMessage(c ronykit.GatewayConn, streamID int64, msg []byte) error {
	err := n.d.Decode(c, msg, n.execute)
	if err != nil {
		return err
	}

	return nil
}

func (n northBridge) execute(e ronykit.Envelope) {}
