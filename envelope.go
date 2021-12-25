package ronykit

type Message interface {
	Unmarshal([]byte) error
	Marshal() ([]byte, error)
}
