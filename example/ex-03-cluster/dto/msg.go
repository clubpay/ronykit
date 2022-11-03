package dto

type SetKeyRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SetKeyResponse struct {
	Success bool `json:"success"`
}

type GetKeyRequest struct {
	Key string `json:"key"`
}

type Key struct {
	Value string `json:"value"`
}
