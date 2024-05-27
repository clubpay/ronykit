package dto

type EchoRequest struct {
	RandomID         int64   `json:"randomID"`
	Ok               bool    `json:"ok"`
	OptionalStrField *string `json:"optionalField"`
	OptionalIntField *int64  `json:"optionalIntField"`
}

type EchoResponse struct {
	RandomID         int64   `json:"randomID"`
	Ok               bool    `json:"ok"`
	OptionalStrField *string `json:"optionalField"`
	OptionalIntField *int64  `json:"optionalIntField"`
}

type SumRequest struct {
	EmbeddedHeader
	Val1 int64 `json:"val1"`
	Val2 int64 `json:"val2"`
}

type SumResponse struct {
	EmbeddedHeader
	Val int64 `json:"val"`
}

type RedirectRequest struct {
	URL string `json:"url"`
}

type EmbeddedHeader struct {
	SomeKey1 string `json:"someKey1"`
	SomeInt1 int64  `json:"someInt1"`
}
