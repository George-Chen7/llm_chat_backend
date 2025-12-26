package controller

// BaseResponse 通用响应结构。
type BaseResponse struct {
	ErrMsg  string `json:"err_msg"`
	ErrCode int    `json:"err_code"`
}
