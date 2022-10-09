package ronykit

// IncomingRPCContainer defines the behavior of RPC message envelopes.
// Basically in RPC communication the actual message should be contained in some kind of container.
// This interface defines a set of guidelines for the implementation of those containers. The user
// of the RonyKIT does not need to use this, and it is basically useful for Bundle developers.
// Although even Bundle developers are not forced to use this interface in their implementation, but
// they are encouraged to.
//
// Example implementations: common.SimpleIncomingJSONRPC
type IncomingRPCContainer interface {
	GetID() string
	// Unmarshal deserialize the received payload.
	Unmarshal(data []byte) error
	// Fill the actual message which will be later used from Context method In().GetMsg().
	Fill(m Message) error
	// GetHdr to read header. This method is used by RonyKIT to fill Envelope's header fields.
	GetHdr(key string) string
	// GetHdrMap returns all the header key-values.
	GetHdrMap() map[string]string
	// Release will be called when we finished our work with it.
	Release()
}

type IncomingRPCFactory func() IncomingRPCContainer

// OutgoingRPCContainer define the behavior of RPC message envelope. Similar to IncomingRPCContainer but
// in another direction.
//
// Example implementations: common.SimpleOutgoingJSONRPC
type OutgoingRPCContainer interface {
	SetID(id string)
	// Marshal serializes the contained message
	Marshal() ([]byte, error)
	// SetHdr set the header.
	SetHdr(k, v string)
	// SetPayload set the body/payload of the container with the actual Message.
	SetPayload(m Message)
	// Release will be called when we finished our work with it.
	Release()
}

type OutgoingRPCFactory func() OutgoingRPCContainer

// Encoding defines the encoding of the messages which will be sent/received. Gateway implementor needs
// to call correct method based on the encoding value.
type Encoding struct {
	tag string
}

func (enc Encoding) Tag() string {
	return enc.tag
}

var (
	Undefined = Encoding{}
	JSON      = Encoding{tag: "json"}
	Proto     = Encoding{tag: "proto"}
	MSG       = Encoding{tag: "msg"}
)

func CustomEncoding(tag string) Encoding {
	return Encoding{tag: tag}
}
