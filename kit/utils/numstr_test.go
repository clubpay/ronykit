package utils_test

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/clubpay/ronykit/kit/utils"
)

var _ = Describe("ParseNumeric tests", func() {
	DescribeTable(
		"inputs of multiple types",
		func(in interface{}, xv float64, xs string) {
			on := utils.ParseNumeric(in)
			Expect(on.Value()).To(BeNumerically("~", xv-1e-6, xv+1e-6))
			Expect(on.String()).To(BeIdenticalTo(xs))
		},
		Entry("string", "13.14", 13.14, "13.14"),
		Entry("float64", float64(13.14), 13.14, "13.14"),
		Entry("float32", float32(13.14), 13.14, "13.14"),
		Entry("int", 13, 13.0, "13.00"),
		Entry("int64", int64(13), 13.0, "13.00"),
	)
})

var _ = Describe("Numeric tests", func() {
	DescribeTable(
		"decoded from json",
		func(str string, xv float64, xs string) {
			type medium struct {
				Field utils.Numeric `json:"f"`
			}
			m := new(medium)

			err := json.Unmarshal([]byte(str), m)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(m.Field.Value()).To(BeNumerically("~", xv-1e-6, xv+1e-6))
			Expect(m.Field.String()).To(BeIdenticalTo(xs))
		},
		Entry("string field", `{"f": "13.14"}`, 13.14, "13.14"),
		Entry("float field", `{"f": 13.14}`, 13.14, "13.14"),
		Entry("int field", `{"f": 13}`, 13.00, "13.00"),
	)

	DescribeTable(
		"with specific precision",
		func(in utils.Numeric, prec int, xv float64, xs string) {
			Expect(in.WithPrecision(prec).Value()).To(BeNumerically("~", xv-1e-6, xv+1e-6))
			Expect(in.WithPrecision(prec).String()).To(BeIdenticalTo(xs))
		},
		Entry("increased", utils.ParseNumeric("13.14"), 3, 13.14, "13.140"),
		Entry("decreased", utils.ParseNumeric("13.14"), 1, 13.14, "13.1"),
		Entry("unchanged", utils.ParseNumeric("13.14"), 2, 13.14, "13.14"),
		Entry("unset", utils.ParseNumeric("13.14"), -1, 13.14, "13.14"),
	)
})
