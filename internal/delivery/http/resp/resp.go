package resp

type Envelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

// 兼容 swagger：用具体类型封装响应体，避免泛型在注释中解析失败。
type AuthLoginEnvelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

type AuthMeEnvelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

type AuthRefreshEnvelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

type APIEnvelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

func OK(data interface{}) Envelope {
	return Envelope{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: "",
	}
}

func Error(code int, msg string) Envelope {
	return Envelope{
		Code:      code,
		Message:   msg,
		Data:      nil,
		RequestID: "",
	}
}

func OKWithRequestID(requestID string, data interface{}) Envelope {
	return Envelope{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestID,
	}
}

func ErrorWithRequestID(requestID string, code int, msg string) Envelope {
	return Envelope{
		Code:      code,
		Message:   msg,
		Data:      nil,
		RequestID: requestID,
	}
}
