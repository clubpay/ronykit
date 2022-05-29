package ronykit_test

import (
	"bytes"
	"io/ioutil"
)

type TestConn struct {
	id       uint64
	clientIP string
	stream   bool
	kv       map[string]string
	buf      *bytes.Buffer
}

func NewTestConn(id uint64, clientIP string, stream bool) TestConn {
	return TestConn{
		id:       id,
		clientIP: clientIP,
		stream:   stream,
		kv:       map[string]string{},
		buf:      &bytes.Buffer{},
	}
}

func (t TestConn) ConnID() uint64 {
	return t.id
}

func (t TestConn) ClientIP() string {
	return t.clientIP
}

func (t TestConn) Write(data []byte) (int, error) {
	return t.buf.Write(data)
}

func (t TestConn) Read() ([]byte, error) {
	return ioutil.ReadAll(t.buf)
}

func (t TestConn) Stream() bool {
	return t.stream
}

func (t TestConn) Walk(f func(key string, val string) bool) {
	for k, v := range t.kv {
		if !f(k, v) {
			return
		}
	}
}

func (t TestConn) Get(key string) string {
	return t.kv[key]
}

func (t TestConn) Set(key string, val string) {
	t.kv[key] = val
}
