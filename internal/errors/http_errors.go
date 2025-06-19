package errors

import (
	"fmt"
)

const (
	HTTP_INVALID_PARAM = "HTTP_INVALID_PARAM"
	HTTP_INVALID_URL   = "HTTP_INVALID_URL"
)

type HttpError struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
	Err    string `json:"err"`
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("%d [%s] | %v", e.Status, e.Code, e.Err)
}

func NewHttpError(code string, status int, err string) *HttpError {
	return &HttpError{
		Code:   code,
		Status: status,
		Err:    err,
	}
}
