package srl_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit/utils/srl"
	"github.com/stretchr/testify/assert"
)

type addressCase struct {
	name string
	addr srl.SRL
	str  string
}

func addressCases() []addressCase {
	return []addressCase{
		{"empty", srl.New("", "", ""), ""},
		{"without group", srl.New("", "", "pos-vendor"), "@pos-vendor"},
		{"without storage & id", srl.New("", "ir", ""), "ir"},
		{"without storage", srl.New("", "ir", "pos-vendor"), "ir@pos-vendor"},
		{"without path & id", srl.New("ws", "", ""), "ws:"},
		{"without path", srl.New("ws", "", "pos-vendor"), "ws:@pos-vendor"},
		{"without id", srl.New("ws", "ir", ""), "ws:ir"},
		{"full", srl.New("ws", "ir", "pos-vendor"), "ws:ir@pos-vendor"},
		{
			"internal worker specific",
			srl.Storage("ws").
				Append(srl.Portable("doshii", "")).
				Append(srl.Portable("str", "")).
				Append(srl.Portable("tr", "")).
				Append(srl.Portable("au", "")).
				Append(srl.Portable("myRestaurant", "myOrder123")),
			"ws:doshii:str:tr:au:myRestaurant@myOrder123",
		},
		{
			"sdk worker specific",
			srl.Storage("ws").
				Append(srl.Portable("doshii", "")).
				Append(srl.Portable("sdk", "")).
				Append(srl.Portable("ds", "")).
				Append(srl.Portable("update-payment-status", "546905640")),
			"ws:doshii:sdk:ds:update-payment-status@546905640",
		},
	}
}

func TestAddressStringer(t *testing.T) {
	for _, tc := range addressCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.str, tc.addr.String())
		})
	}
}

func TestAddressParser(t *testing.T) {
	for _, tc := range addressCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.addr, srl.Parse(tc.str))
		})
	}
}

func TestParseFormattedAddress(t *testing.T) {
	for _, tc := range addressCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.addr, srl.Parse(tc.addr.String()))
		})
	}
}

func TestFormatParsedAddress(t *testing.T) {
	for _, tc := range addressCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.str, srl.Parse(tc.str).String())
		})
	}
}

type binaryCase struct {
	name     string
	addr1    srl.SRL
	addr2    srl.SRL
	expected srl.SRL
}

func appendCases() []binaryCase {
	return []binaryCase{
		{"both empty", srl.New("", "", ""), srl.New("", "", ""), srl.New("", "", "")},
		{"q with storage, empty src", srl.New("ws", "", ""), srl.New("", "", ""), srl.New("ws", "", "")},
		{"empty q, src with storage", srl.New("", "", ""), srl.New("ws", "", ""), srl.New("ws", "", "")},
		{"q & src with storage", srl.New("ws", "", ""), srl.New("rm", "", ""), srl.New("rm", "", "")},
		{"full q, empty src", srl.New("ws", "ir", "sapaad"), srl.New("", "", ""), srl.New("ws", "ir", "sapaad")},
		{"empty q, full src", srl.New("", "", ""), srl.New("ws", "ir", "sapaad"), srl.New("ws", "ir", "sapaad")},
		{"q with path & id, src with storage", srl.New("", "ir", "sapaad"), srl.New("ws", "", ""), srl.New("ws", "ir", "sapaad")},
		{"q with storage, src with path & id", srl.New("ws", "", ""), srl.New("", "ir", "sapaad"), srl.New("ws", "ir", "sapaad")},
		{"full q & src", srl.New("ws", "ir", "sapaad"), srl.New("rm", "us", "foodics"), srl.New("rm", "ir:us", "foodics")},
	}
}

func replaceCases() []binaryCase {
	return []binaryCase{
		{"both empty", srl.New("", "", ""), srl.New("", "", ""), srl.New("", "", "")},
		{"q with storage, empty src", srl.New("ws", "", ""), srl.New("", "", ""), srl.New("", "", "")},
		{"empty q, src with storage", srl.New("", "", ""), srl.New("ws", "", ""), srl.New("ws", "", "")},
		{"q & src with storage", srl.New("ws", "", ""), srl.New("rm", "", ""), srl.New("rm", "", "")},
		{"full q, empty src", srl.New("ws", "ir", "sapaad"), srl.New("", "", ""), srl.New("", "", "")},
		{"empty q, full src", srl.New("", "", ""), srl.New("ws", "ir", "sapaad"), srl.New("ws", "ir", "sapaad")},
		{"q with path & id, src with storage", srl.New("", "ir", "sapaad"), srl.New("ws", "", ""), srl.New("ws", "", "")},
		{"q with storage, src with path & id", srl.New("ws", "", ""), srl.New("", "ir", "sapaad"), srl.New("", "ir", "sapaad")},
		{"full q & src", srl.New("ws", "ir", "sapaad"), srl.New("rm", "us", "foodics"), srl.New("rm", "us", "foodics")},
	}
}

func mergeCases() []binaryCase {
	return []binaryCase{
		{"both empty", srl.New("", "", ""), srl.New("", "", ""), srl.New("", "", "")},
		{"q with storage, empty src", srl.New("ws", "", ""), srl.New("", "", ""), srl.New("ws", "", "")},
		{"empty q, src with storage", srl.New("", "", ""), srl.New("ws", "", ""), srl.New("ws", "", "")},
		{"q & src with storage", srl.New("ws", "", ""), srl.New("rm", "", ""), srl.New("rm", "", "")},
		{"full q, empty src", srl.New("ws", "ir", "sapaad"), srl.New("", "", ""), srl.New("ws", "ir", "sapaad")},
		{"empty q, full src", srl.New("", "", ""), srl.New("ws", "ir", "sapaad"), srl.New("ws", "ir", "sapaad")},
		{"q with path & id, src with storage", srl.New("", "ir", "sapaad"), srl.New("ws", "", ""), srl.New("ws", "ir", "sapaad")},
		{"q with storage, src with path & id", srl.New("ws", "", ""), srl.New("", "ir", "sapaad"), srl.New("ws", "ir", "sapaad")},
		{"full q & src", srl.New("ws", "ir", "sapaad"), srl.New("rm", "us", "foodics"), srl.New("rm", "us", "foodics")},
	}
}

func TestAddressAppend(t *testing.T) {
	for _, tc := range appendCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.addr1.Append(tc.addr2))
		})
	}
}

func TestAddressReplace(t *testing.T) {
	for _, tc := range replaceCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.addr1.Replace(tc.addr2))
		})
	}
}

func TestAddressMerge(t *testing.T) {
	for _, tc := range mergeCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.addr1.Merge(tc.addr2))
		})
	}
}
