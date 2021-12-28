package main

import (
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit/exmples/simple-jsonrpc/msg"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
)

func main() {
	c, err := gnet.NewClient(&eventHandler{})
	if err != nil {
		panic(err)
	}

	c.Start()
	defer c.Stop()

	conn, err := c.Dial("tcp4", "127.0.0.1:80")
	if err != nil {
		panic(err)
	}

	req := &msg.EchoRequest{
		RandomID: 120,
	}
	reqBytes, _ := req.Marshal()
	envelope := jsonrpc.Envelope{
		Predicate: "echoRequest",
		ID:        1,
		Payload:   reqBytes,
	}
	envelopeBytes, _ := json.Marshal(envelope)
	err = conn.AsyncWrite(envelopeBytes)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)
}

type eventHandler struct {
	gnet.EventServer
}

func (eh *eventHandler) React(packet []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	fmt.Println(string(packet))

	return nil, gnet.None
}
