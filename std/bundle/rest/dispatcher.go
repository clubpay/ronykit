package rest

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type Dispatcher struct{}

func (d Dispatcher) Serialize(conn ronykit.Conn, streamID int64, envelopes ...ronykit.Envelope) error {
	if _, ok := conn.(ronykit.REST); !ok {
		return ErrConnectionIsNoREST
	}

	switch len(envelopes) {
	case 0:
	case 1:
		jsonData, err := envelopes[0].GetMessage().Marshal()
		if err != nil {
			return err
		}

		return conn.Write(streamID, jsonData)
	default:
		container := &Container{}
		for _, e := range envelopes {
			container.Messages = append(container.Messages, e.GetMessage().(*Message))
		}

		jsonData, err := container.Marshal()
		if err != nil {
			return err
		}

		return conn.Write(streamID, jsonData)
	}

	return nil
}

func (d Dispatcher) Deserialize(conn ronykit.Conn, data []byte, f func(envelope ronykit.Envelope) error) error {
	rc, ok := conn.(ronykit.REST)
	if !ok {
		return ErrConnectionIsNoREST
	}

	e := envelopePool.Get()
	e.HDR[Path] = rc.GetPath()
	e.HDR[Method] = rc.GetMethod()

	err := json.Unmarshal(data, e)
	if err != nil {
		return err
	}

	err = f(e)
	envelopePool.Release(e)

	return err
}

func (d Dispatcher) OnOpen(conn ronykit.Conn) {
	//TODO implement me
	panic("implement me")
}

var (
	ErrConnectionIsNoREST = fmt.Errorf("connection is not REST")
	ErrRouteNotFound      = fmt.Errorf("route not found")
)
