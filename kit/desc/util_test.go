package desc_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/stretchr/testify/assert"
)

type customStruct struct {
	Param1      string          `json:"param1"`
	Param2      int64           `json:"param2"`
	Obj1        customSubStruct `json:"obj1"`
	Obj2        customInterface `json:"obj2"`
	PtrParam3   *string         `json:"ptrParam3"`
	PtrSubParam *customStruct   `json:"prtSubParam"`
	RawJSON     json.RawMessage `json:"rawJSON"`
}

type customSubStruct struct {
	SubParam1   string                      `json:"subParam1"`
	SubParam2   int                         `json:"subParam2"`
	MapParam    map[string]anotherSubStruct `json:"mapParam"`
	MapPtrParam map[int]*anotherSubStruct   `json:"mapPtr"`
}

type anotherSubStruct struct {
	Keys       []string         `json:"keys"`
	Values     map[int64]string `json:"values"`
	Interfaces map[string]any   `json:"interfaces"`
}

type customInterface interface {
	Method1() string
	Method2() customStruct
}

func TestCheckType(t *testing.T) {
	assert.Equal(t, "customStruct", desc.TypeOf("", reflect.TypeOf(customStruct{})))
	assert.Equal(t, "*customInterface", desc.TypeOf("", reflect.TypeOf(new(customInterface))))
	assert.Equal(t, "int64", desc.TypeOf("", reflect.TypeOf(int64(0))))
	assert.Equal(t, "[]int64", desc.TypeOf("", reflect.TypeOf([]int64{})))
	assert.Equal(t, "[18]int64", desc.TypeOf("", reflect.TypeOf([18]int64{})))
	assert.Equal(t, "map[string]*customStruct", desc.TypeOf("", reflect.TypeOf(map[string]*customStruct{})))
	assert.Equal(t, "map[string]any", desc.TypeOf("", reflect.TypeOf(map[string]any{})))
}
