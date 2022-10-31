package ronykit

import "github.com/goccy/go-json"

// envelopeCarrier is a serializable message which is used by Cluster component of the
// EdgeServer to send information from one instance to another instance.
type envelopeCarrier struct {
	ConnHdr     map[string]string `json:"connHdr"`
	Hdr         map[string]string `json:"hdr"`
	Msg         []byte            `json:"msg"`
	ContractID  string            `json:"cid"`
	ServiceName string            `json:"svc"`
	ExecIndex   int               `json:"idx"`
	IsREST      bool              `json:"isRest"`
}

func envelopeCarrierFromContext(ctx *Context) envelopeCarrier {
	ec := envelopeCarrier{
		Hdr:         map[string]string{},
		ConnHdr:     map[string]string{},
		Msg:         nil,
		ContractID:  ctx.ContractID(),
		ServiceName: ctx.ServiceName(),
		ExecIndex:   ctx.handlerIndex,
		IsREST:      ctx.isREST(),
	}

	ec.Msg, _ = MarshalMessage(ctx.In().GetMsg())
	ctx.In().
		WalkHdr(func(key, val string) bool {
			ec.Hdr[key] = val

			return true
		})
	ctx.Conn().Walk(func(key string, val string) bool {
		ec.ConnHdr[key] = val

		return true
	})

	return ec
}

func envelopeCarrierFromData(data []byte) envelopeCarrier {
	ec := envelopeCarrier{}
	_ = json.UnmarshalNoEscape(data, &ec)

	return ec
}
