package stub_test

import (
	"github.com/clubpay/ronykit/stub"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var keyValues = func(in string) string {
	switch in {
	case "p1":
		return "value1"
	case "p2":
		return "value2"
	case "p3":
		return "value3"
	}

	return in
}

var _ = DescribeTable(
	"Fill URL Params",
	func(in, out string, f func(string) string) {
		Expect(stub.FillParams(in, f)).To(Equal(out))
	},
	Entry("",
		"/some/{p1}/{p2}?something={p3}&boolean",
		"/some/value1/value2?something=value3&boolean",
		keyValues,
	),
	Entry("",
		"/some/{p1}/{p2}?something={p3}&boolean",
		"/some/value1/value2?something=value3&boolean",
		keyValues,
	),
	Entry("",
		"/some/{p1}{p2}/?something={p3}&boolean",
		"/some/value1value2/?something=value3&boolean",
		keyValues,
	),
	Entry("",
		"/some/{p1}",
		"/some/value1",
		keyValues,
	),
)
