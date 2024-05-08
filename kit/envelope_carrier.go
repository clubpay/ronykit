package kit

import (
	"encoding/json"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/goccy/go-reflect"
)

type carrierKind int

const (
	incomingCarrier carrierKind = iota + 1
	outgoingCarrier
	eofCarrier
)

// envelopeCarrier is a serializable message which is used by Cluster component of the
// EdgeServer to send information from one instance to another instance.
type envelopeCarrier struct {
	// SessionID is a unique identifier for each remote-execution session.
	SessionID string `json:"id"`
	// Kind identifies what type of the data this carrier has
	Kind carrierKind `json:"kind"`
	// OriginID the instance's id of the sender of this message
	OriginID string `json:"originID"`
	// TargetID the instance's id of the receiver of this message
	TargetID string       `json:"targetID"`
	Data     *carrierData `json:"data"`
}

func (ec *envelopeCarrier) FillWithContext(ctx *Context) *envelopeCarrier {
	ec.Data = &carrierData{
		EnvelopeID:  ctx.In().GetID(),
		IsREST:      ctx.IsREST(),
		MsgType:     reflect.TypeOf(ctx.In().GetMsg()).String(),
		Msg:         marshalEnvelopeCarrier(ctx.In().GetMsg()),
		ContractID:  ctx.ContractID(),
		ServiceName: ctx.ServiceName(),
		Route:       ctx.Route(),
		ConnHdr:     map[string]string{},
		Hdr:         map[string]string{},
	}

	if tp := ctx.sb.tp; tp != nil {
		tp.Inject(ctx.ctx, ec.Data)
	}

	ctx.In().
		WalkHdr(func(key, val string) bool {
			ec.Data.Hdr[key] = val

			return true
		})

	ctx.Conn().
		Walk(func(key string, val string) bool {
			ec.Data.ConnHdr[key] = val

			return true
		})

	return ec
}

func (ec *envelopeCarrier) FillWithEnvelope(e *Envelope) *envelopeCarrier {
	ec.Data = &carrierData{
		EnvelopeID:  utils.B2S(e.id),
		IsREST:      e.ctx.IsREST(),
		MsgType:     reflect.TypeOf(e.GetMsg()).String(),
		Msg:         marshalEnvelopeCarrier(e.GetMsg()),
		ContractID:  e.ctx.ContractID(),
		ServiceName: e.ctx.ServiceName(),
		Route:       e.ctx.Route(),
		ConnHdr:     map[string]string{},
		Hdr:         map[string]string{},
	}

	e.WalkHdr(func(key string, val string) bool {
		ec.Data.Hdr[key] = val

		return true
	})

	e.conn.Walk(func(key string, val string) bool {
		ec.Data.ConnHdr[key] = val

		return true
	})

	return ec
}

type carrierData struct {
	EnvelopeID  string            `json:"envelopID,omitempty"`
	ConnHdr     map[string]string `json:"connHdr,omitempty"`
	IsREST      bool              `json:"isREST,omitempty"`
	Hdr         map[string]string `json:"hdr,omitempty"`
	MsgType     string            `json:"msgType,omitempty"`
	Msg         []byte            `json:"msg,omitempty"`
	ContractID  string            `json:"cid,omitempty"`
	ServiceName string            `json:"svc,omitempty"`
	Route       string            `json:"route,omitempty"`
}

func (c carrierData) Get(key string) string {
	return c.ConnHdr[key]
}

func (c carrierData) Set(key string, value string) {
	c.ConnHdr[key] = value
}

func (ec *envelopeCarrier) ToJSON() []byte {
	data, _ := json.Marshal(ec)

	return data
}

func (ec *envelopeCarrier) FromJSON(data []byte) error {
	err := json.Unmarshal(data, ec)
	if err != nil {
		return err
	}

	return nil
}

func newEnvelopeCarrier(kind carrierKind, sessionID, originID, targetID string) *envelopeCarrier {
	return &envelopeCarrier{
		Kind:      kind,
		SessionID: sessionID,
		OriginID:  originID,
		TargetID:  targetID,
	}
}

func unmarshalEnvelopeCarrier(data []byte, m Message) {
	err := json.Unmarshal(data, m)
	if err != nil {
		panic(err)
	}
}

func marshalEnvelopeCarrier(m Message) []byte {
	switch v := m.(type) {
	case RawMessage:
		return v
	}
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return data
}
