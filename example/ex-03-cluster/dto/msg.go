package dto

type EchoRequest struct {
	RandomID int64 `json:"randomID"`
}

type EchoResponse struct {
	RandomID int64  `json:"randomID"`
	OriginID string `json:"origin"`
}
