package dto

import "encoding/json"

type VeryComplexRequest struct {
	Key1      string             `json:"key1"`
	Key1Ptr   *string            `json:"key1Ptr"`
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
	Key1      string              `json:"key1"`
	Key1Ptr   *string             `json:"key1Ptr"`
	MapKey1   map[string]int      `json:"mapKey1"`
	MapKey2   map[int64]*KeyValue `json:"mapKey2"`
	SliceKey1 []byte              `json:"sliceKey1"`
	SliceKey2 []KeyValue          `json:"sliceKey2"`
}
