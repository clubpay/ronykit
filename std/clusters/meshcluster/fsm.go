package meshcluster

import (
	"io"
	"time"

	"github.com/goccy/go-json"
	"github.com/hashicorp/raft"
)

type fsm struct {
	db *badgerKVStore
}

var _ raft.FSM = (*fsm)(nil)

func newFSM(db *badgerKVStore) *fsm {
	return &fsm{
		db: db,
	}
}

func (f fsm) Apply(log *raft.Log) interface{} {
	var msgEnvelope raftMsgEnvelope
	err := json.Unmarshal(log.Data, msgEnvelope)
	if err != nil {
		return raftMsgEnvelope{
			Error: &raftMsgError{
				Msg: err.Error(),
			},
		}
	}

	switch msgEnvelope.Code {
	case setKeyRequestMsg:
		var msg setKeyRequest
		_ = json.Unmarshal(msgEnvelope.Msg, &msg)
		_ = f.db.SetTTL(msg.Key, msg.Value, msg.TTL)
	case getKeyRequestMsg:
		var msg getKeyRequest
		_ = json.Unmarshal(msgEnvelope.Msg, &msg)
		_, _ = f.db.Get(msg.Key)
	case delKeyRequestMsg:
		var msg delKeyRequest
		_ = json.Unmarshal(msgEnvelope.Msg, &msg)
		_ = f.db.Del(msg.Key)
	}

	return nil
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	// TODO implement me
	panic("implement me")
}

func (f fsm) Restore(snapshot io.ReadCloser) error {
	// TODO implement me
	panic("implement me")
}

type raftMsgCode int

const (
	setKeyRequestMsg raftMsgCode = iota
	getKeyRequestMsg
	delKeyRequestMsg
)

type raftMsgEnvelope struct {
	Code  raftMsgCode
	Error *raftMsgError
	Msg   json.RawMessage
}

type raftMsgError struct {
	Code int
	Msg  string
}

type setKeyRequest struct {
	Key   []byte
	Value []byte
	TTL   time.Duration
}

type getKeyRequest struct {
	Key []byte
}

type delKeyRequest struct {
	Key []byte
}
