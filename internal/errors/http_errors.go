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
	Status int    `json:"-"`
	Err    error  `json:"err"`
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("[%s] %d: %v", e.Code, e.Status, e.Err)
}

func (e *HttpError) Unwrap() error {
	return e.Err
}

func NewHttpError(code string, status int, err error) *HttpError {
	return &HttpError{
		Code:   code,
		Status: status,
		Err:    err,
	}
}
