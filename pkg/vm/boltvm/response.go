package boltvm

type Response struct {
	Ok     bool
	Result []byte
}

// Result returns normal result
func Success(data []byte) *Response {
	return &Response{
		Ok:     true,
		Result: data,
	}
}

// Error returns error result that will cause
// vm call error, and this transaction will be invalid
func Error(msg string) *Response {
	return &Response{
		Ok:     false,
		Result: []byte(msg),
	}
}
