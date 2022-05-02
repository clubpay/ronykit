package msg

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EchoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
}

type EchoResponse struct {
	RandomID int64 `json:"randomId"`
}

type SumRequest struct {
	Val1 int64
	Val2 int64
}

type SumResponse struct {
	Val int64
}
