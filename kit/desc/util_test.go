package desc_test

import (
	"reflect"

	"github.com/clubpay/ronykit/kit/desc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type customStruct struct {
	Param1      string          `json:"param1"`
	Param2      int64           `json:"param2"`
	Obj1        customSubStruct `json:"obj1"`
	Obj2        customInterface `json:"obj2"`
	PtrParam3   *string         `json:"ptrParam3"`
	PrtSubParam *customStruct   `json:"prtSubParam"`
}

type customSubStruct struct {
	SubParam1   string                      `json:"subParam1"`
	SubParam2   int                         `json:"subParam2"`
	MapParam    map[string]anotherSubStruct `json:"mapParam"`
	MapPtrParam map[int]*anotherSubStruct   `json:"mapPtr"`
}

type anotherSubStruct struct {
	Keys       []string               `json:"keys"`
	Values     map[int64]string       `json:"values"`
	Interfaces map[string]interface{} `json:"interfaces"`
}

type customInterface interface {
	Method1() string
	Method2() customStruct
}

var _ = Describe("Check Type", func() {
	It("should detect types correctly", func() {
		Expect(desc.TypeOf("", reflect.TypeOf(customStruct{}))).To(Equal("customStruct"))
		Expect(desc.TypeOf("", reflect.TypeOf(new(customInterface)))).To(Equal("*customInterface"))
		Expect(desc.TypeOf("", reflect.TypeOf(int64(0)))).To(Equal("int64"))
		Expect(desc.TypeOf("", reflect.TypeOf([]int64{}))).To(Equal("[]int64"))
		Expect(desc.TypeOf("", reflect.TypeOf([18]int64{}))).To(Equal("[18]int64"))
		Expect(desc.TypeOf("", reflect.TypeOf(map[string]*customStruct{}))).To(Equal("map[string]*customStruct"))
		Expect(desc.TypeOf("", reflect.TypeOf(map[string]interface{}{}))).To(Equal("map[string]interface{}"))
	})
})
