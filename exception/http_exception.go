package exception

type HttpException struct {
	Code    int
	Message string
}

func (e *HttpException) Error() string {
	return e.Message
}

func NewHttpException(code int, message string) *HttpException {
	return &HttpException{
		Code:    code,
		Message: message,
	}
}
