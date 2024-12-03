package util

type Response struct {
	Error string `json:"error,omitempty"`
	Data  any    `json:"data"`
}

func NewResponse(data any) *Response {
	return &Response{Data: data}
}

func NewErrorResponse(err error) *Response {
	return &Response{Error: err.Error()}
}
