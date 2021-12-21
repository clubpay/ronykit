package edge

import (
	"github.com/ronaksoft/ronykit"
)

type gatewayDelegate struct{}

func (g gatewayDelegate) OnOpen(c ronykit.GatewayConn) {

	//TODO implement me
	panic("implement me")
}

func (g gatewayDelegate) OnClose(connID uint64) {
	//TODO implement me
	panic("implement me")
}

func (g gatewayDelegate) OnMessage(c ronykit.GatewayConn, streamID int64, msg []byte) {
	//TODO implement me
	panic("implement me")
}
