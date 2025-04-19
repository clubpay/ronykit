package dto

import (
	"encoding/json"
	"time"
)

type SimpleHdr struct {
	Key1 string      `json:"sKey1"`
	Key2 int         `json:"sKey2"`
	T1   time.Time   `json:"t1"`
	T2   *time.Time  `json:"t2"`
	T3   []time.Time `json:"t3"`
}

type VeryComplexRequest struct {
	SimpleHdr
	Key1      string             `json:"key1"`
	Key1Ptr   *string            `json:"key1Ptr"`
	Key2Ptr   *int               `json:"key2Ptr,omitempty"`
	MapKey1   map[string]int     `json:"mapKey1"`
	MapKey2   map[int64]KeyValue `json:"mapKey2"`
	SliceKey1 []bool             `json:"sliceKey1"`
	SliceKey2 []*KeyValue        `json:"sliceKey2"`
	RawKey    json.RawMessage    `json:"rawKey"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

type VeryComplexResponse struct {
	Key1      string              `json:"key1,omitempty"`
	Key1Ptr   *string             `json:"key1Ptr,omitempty"`
	MapKey1   map[string]int      `json:"mapKey1,omitempty"`
	MapKey2   map[int64]*KeyValue `json:"mapKey2,omitempty"`
	SliceKey1 []byte              `json:"sliceKey1"`
	SliceKey2 []KeyValue          `json:"sliceKey2"`
}
