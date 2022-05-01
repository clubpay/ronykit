package dto

type EchoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
	Ok       bool  `json:"ok" paramName:"ok"`
}

type EchoResponse struct {
	RandomID int64 `json:"randomId"`
	Ok       bool  `json:"ok"`
}

type SumRequest struct {
	Val1 int64 `paramName:"val1" json:"val1"`
	Val2 int64 `paramName:"val2" json:"val2"`
}

type SumResponse struct {
	Val int64
}

type RedirectRequest struct {
	URL string `json:"url" paramName:"url"`
}
