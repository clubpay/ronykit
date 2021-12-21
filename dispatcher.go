package ronykit

type Dispatcher interface {
	// Serialize will be called on the outgoing messages to encode them into the connection.
	// it is responsible for write data to conn. This is the implementation specific to send
	// each Envelope separately or compact them in one other envelope.
	Serialize(conn Conn, streamID int64, envelopes ...Envelope) error
	// Deserialize decodes the incoming wire messages, and it may decode into one or more envelopes
	// Implementer can call f sequentially or concurrently, but it MUST not return until all the
	// call are returned.
	// NOTE: Implementer can re-use the Envelope when the function 'f' is returned.
	Deserialize(conn Conn, data []byte, f func(envelope Envelope) error) error
	// OnOpen will be called when a new connection has been opened
	OnOpen(conn Conn)
}
