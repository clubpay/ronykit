package main

type echoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
	Ok       bool  `json:"ok" paramName:"ok"`
}

type echoResponse struct {
	RandomID int64 `json:"randomId"`
	Ok       bool  `json:"ok"`
}

type sumRequest struct {
	Val1 int64 `paramName:"val1" json:"val1"`
	Val2 int64 `paramName:"val2" json:"val2"`
}

type sumResponse struct {
	Val int64
}
