package ronykit

import "github.com/goccy/go-json"

type carrierKind int

const (
	outgoingCarrier carrierKind = iota + 1
	incomingCarrier
	eofCarrier
)

// envelopeCarrier is a serializable message which is used by Cluster component of the
// EdgeServer to send information from one instance to another instance.
type envelopeCarrier struct {
	Kind        carrierKind       `json:"kind"`
	ID          string            `json:"id"`
	ConnHdr     map[string]string `json:"connHdr,omitempty"`
	IsREST      bool              `json:"isREST,omitempty"`
	Hdr         map[string]string `json:"hdr,omitempty"`
	Msg         Message           `json:"msg,omitempty"`
	ContractID  string            `json:"cid,omitempty"`
	ServiceName string            `json:"svc,omitempty"`
	ExecIndex   int               `json:"idx,omitempty"`
	Route       string            `json:"route,omitempty"`
}

func (ec *envelopeCarrier) ToJSON() []byte {
	data, _ := json.MarshalNoEscape(ec)

	return data
}

func envelopeCarrierFromContext(ctx *Context, kind carrierKind) *envelopeCarrier {
	ec := &envelopeCarrier{
		ID:          ctx.In().GetID(),
		Kind:        kind,
		ConnHdr:     map[string]string{},
		Hdr:         map[string]string{},
		IsREST:      ctx.isREST(),
		Msg:         ctx.In().GetMsg(),
		ContractID:  ctx.ContractID(),
		ServiceName: ctx.ServiceName(),
		ExecIndex:   ctx.handlerIndex,
		Route:       ctx.Route(),
	}

	ctx.In().
		WalkHdr(func(key, val string) bool {
			ec.Hdr[key] = val

			return true
		})

	ctx.Conn().
		Walk(func(key string, val string) bool {
			ec.ConnHdr[key] = val

			return true
		})

	return ec
}

func envelopeCarrierFromData(data []byte) (*envelopeCarrier, error) {
	ec := &envelopeCarrier{}
	err := UnmarshalMessage(data, ec)
	if err != nil {
		return nil, err
	}

	return ec, nil
}
