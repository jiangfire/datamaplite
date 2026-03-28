package response

const (
	SuccessCode = 0
)

// HttpResult 标准API响应结构
type HttpResult struct {
	Code      int         `json:"code"`                 // 业务状态码，成功固定为0
	Message   string      `json:"message,omitempty"`    // 响应消息
	ErrorCode string      `json:"error_code,omitempty"` // 业务错误码
	Data      interface{} `json:"data,omitempty"`       // 响应数据
}

func Success(data interface{}) HttpResult {
	return HttpResult{
		Code: SuccessCode,
		Data: data,
	}
}

func SuccessWithMessage(message string, data interface{}) HttpResult {
	return HttpResult{
		Code:    SuccessCode,
		Message: message,
		Data:    data,
	}
}

func Error(code int, errorCode string, message string) HttpResult {
	return HttpResult{
		Code:      code,
		Message:   message,
		ErrorCode: errorCode,
	}
}
