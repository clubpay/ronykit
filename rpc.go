package ronykit

// IncomingRPCContainer defines the behavior of RPC message envelopes.
// Basically in RPC communication the actual message should be contained in some kind of container.
// This interface defines a set of guidelines for the implementation of those containers. The user
// of the RonyKIT does not need to use this, and it is basically useful for Gateway developers.
// Although even Gateway developers are not forced to use this interface in their implementation but
// they are encouraged to.
//
// Example implementations: common.SimpleIncomingJSONRPC
type IncomingRPCContainer interface {
	Unmarshal(data []byte) error
	Fill(m Message) error
	GetHdr(key string) string
	GetHdrMap() map[string]string
}

type IncomingRPCFactory func() IncomingRPCContainer

// OutgoingRPCContainer define the behavior of RPC message envelope. Similar to IncomingRPCContainer but
// in another direction.
//
// Example implementations: common.SimpleOutgoingJSONRPC
type OutgoingRPCContainer interface {
	Marshal() ([]byte, error)
	SetHdr(k, v string)
	SetPayload(m Message)
	Reset()
}

type OutgoingRPCFactory func() OutgoingRPCContainer
