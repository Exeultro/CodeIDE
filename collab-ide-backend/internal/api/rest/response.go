package rest

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code,omitempty"`
}

func Ok(data interface{}) Response {
	return Response{Success: true, Data: data}
}

func Fail(code int, errMsg string) Response {
	return Response{Success: false, Error: errMsg, Code: code}
}
