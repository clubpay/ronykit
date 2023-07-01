package srl_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit/utils/srl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSRL(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "SRL TestSuite")
}

var _ = Describe("Address stringer & formatter", func() {
	entries := []any{
		Entry("empty", srl.New("", "", ""), ""),
		Entry("without group", srl.New("", "", "pos-vendor"), "@pos-vendor"),
		Entry("without storage & id", srl.New("", "ir", ""), "ir"),
		Entry("without storage", srl.New("", "ir", "pos-vendor"), "ir@pos-vendor"),
		Entry("without path & id", srl.New("ws", "", ""), "ws:"),
		Entry("without path", srl.New("ws", "", "pos-vendor"), "ws:@pos-vendor"),
		Entry("without id", srl.New("ws", "ir", ""), "ws:ir"),
		Entry("full", srl.New("ws", "ir", "pos-vendor"), "ws:ir@pos-vendor"),
		Entry(
			"internal worker specific",
			srl.Storage("ws").
				Append(srl.Portable("doshii", "")).
				Append(srl.Portable("str", "")).
				Append(srl.Portable("tr", "")).
				Append(srl.Portable("au", "")).
				Append(srl.Portable("myRestaurant", "myOrder123")),
			"ws:doshii:str:tr:au:myRestaurant@myOrder123",
		),
		Entry(
			"sdk worker specific",
			srl.Storage("ws").
				Append(srl.Portable("doshii", "")).
				Append(srl.Portable("sdk", "")).
				Append(srl.Portable("ds", "")).
				Append(srl.Portable("update-payment-status", "546905640")),
			"ws:doshii:sdk:ds:update-payment-status@546905640",
		),
	}

	DescribeTable(
		"address stringer",
		append(
			[]any{
				func(addr srl.SRL, str string) {
					Expect(addr.String()).To(BeIdenticalTo(str))
				},
			}, entries...)...,
	)

	DescribeTable(
		"address parser",
		append(
			[]any{
				func(addr srl.SRL, str string) {
					Expect(srl.Parse(str)).To(BeIdenticalTo(addr))
				},
			}, entries...)...,
	)

	DescribeTable(
		"parse formatted address",
		append(
			[]any{
				func(addr srl.SRL, _ string) {
					Expect(srl.Parse(addr.String())).To(BeIdenticalTo(addr))
				},
			}, entries...)...,
	)

	DescribeTable(
		"format parsed address",
		append(
			[]any{
				func(_ srl.SRL, str string) {
					Expect(srl.Parse(str).String()).To(BeIdenticalTo(str))
				},
			}, entries...)...,
	)
})

// nolint
var _ = Describe("Address append", func() {
	entries := []any{
		Entry("both empty",
			srl.New("", "", ""),
			srl.New("", "", ""),
			srl.New("", "", ""),
		),
		Entry("q with storage, empty src",
			srl.New("ws", "", ""),
			srl.New("", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("empty q, src with storage",
			srl.New("", "", ""),
			srl.New("ws", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("q & src with storage",
			srl.New("ws", "", ""),
			srl.New("rm", "", ""),
			srl.New("rm", "", ""),
		),

		Entry("full q, empty src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("", "", ""),
			srl.New("ws", "ir", "sapaad"),
		),
		Entry("empty q, full src",
			srl.New("", "", ""),
			srl.New("ws", "ir", "sapaad"),
			srl.New("ws", "ir", "sapaad"),
		),

		Entry("q with path & id, src with storage",
			srl.New("", "ir", "sapaad"),
			srl.New("ws", "", ""),
			srl.New("ws", "ir", "sapaad"),
		),
		Entry("q with storage, src with path & id",
			srl.New("ws", "", ""),
			srl.New("", "ir", "sapaad"),
			srl.New("ws", "ir", "sapaad"),
		),

		Entry("full q & src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("rm", "us", "foodics"),
			srl.New("rm", "ir:us", "foodics"),
		),
	}

	DescribeTable(
		"append two addresses",
		append(
			[]any{
				func(addr1, addr2, expected srl.SRL) {
					Expect(addr1.Append(addr2)).To(BeIdenticalTo(expected))
				},
			}, entries...)...,
	)
})

// nolint
var _ = Describe("Address replace", func() {
	entries := []any{
		Entry("both empty",
			srl.New("", "", ""),
			srl.New("", "", ""),
			srl.New("", "", ""),
		),
		Entry("q with storage, empty src",
			srl.New("ws", "", ""),
			srl.New("", "", ""),
			srl.New("", "", ""),
		),
		Entry("empty q, src with storage",
			srl.New("", "", ""),
			srl.New("ws", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("q & src with storage",
			srl.New("ws", "", ""),
			srl.New("rm", "", ""),
			srl.New("rm", "", ""),
		),

		Entry("full q, empty src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("", "", ""),
			srl.New("", "", ""),
		),
		Entry("empty q, full src",
			srl.New("", "", ""),
			srl.New("ws", "ir", "sapaad"),
			srl.New("ws", "ir", "sapaad"),
		),

		Entry("q with path & id, src with storage",
			srl.New("", "ir", "sapaad"),
			srl.New("ws", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("q with storage, src with path & id",
			srl.New("ws", "", ""),
			srl.New("", "ir", "sapaad"),
			srl.New("", "ir", "sapaad"),
		),

		Entry("full q & src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("rm", "us", "foodics"),
			srl.New("rm", "us", "foodics"),
		),
	}

	DescribeTable(
		"replace two addresses",
		append(
			[]any{
				func(addr1, addr2, expected srl.SRL) {
					Expect(addr1.Replace(addr2)).To(BeIdenticalTo(expected))
				},
			}, entries...)...,
	)
})

// nolint
var _ = Describe("Address merge", func() {
	entries := []any{
		Entry("both empty",
			srl.New("", "", ""),
			srl.New("", "", ""),
			srl.New("", "", ""),
		),
		Entry("q with storage, empty src",
			srl.New("ws", "", ""),
			srl.New("", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("empty q, src with storage",
			srl.New("", "", ""),
			srl.New("ws", "", ""),
			srl.New("ws", "", ""),
		),
		Entry("q & src with storage",
			srl.New("ws", "", ""),
			srl.New("rm", "", ""),
			srl.New("rm", "", ""),
		),

		Entry("full q, empty src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("", "", ""),
			srl.New("ws", "ir", "sapaad"),
		),
		Entry("empty q, full src",
			srl.New("", "", ""),
			srl.New("ws", "ir", "sapaad"),
			srl.New("ws", "ir", "sapaad"),
		),

		Entry("q with path & id, src with storage",
			srl.New("", "ir", "sapaad"),
			srl.New("ws", "", ""),
			srl.New("ws", "ir", "sapaad"),
		),
		Entry("q with storage, src with path & id",
			srl.New("ws", "", ""),
			srl.New("", "ir", "sapaad"),
			srl.New("ws", "ir", "sapaad"),
		),

		Entry("full q & src",
			srl.New("ws", "ir", "sapaad"),
			srl.New("rm", "us", "foodics"),
			srl.New("rm", "us", "foodics"),
		),
	}

	DescribeTable(
		"merge two addresses",
		append(
			[]any{
				func(addr1, addr2, expected srl.SRL) {
					Expect(addr1.Merge(addr2)).To(BeIdenticalTo(expected))
				},
			}, entries...)...,
	)
})
