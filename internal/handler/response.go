package handler

type BaseResponse struct {
	ErrMsg  string `json:"err_msg"`
	ErrCode int    `json:"err_code"`
}

type SuccessResponse[T any] struct {
	BaseResponse
	Data T `json:"data"`
}
