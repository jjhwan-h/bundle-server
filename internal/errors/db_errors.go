package errors

import "fmt"

type DBError struct {
	Code    string
	Message string
	Err     error
}

const (
	DB_INIT_FAIL  = "DB_INIT_FAIL"
	DB_QUERY_FAIL = "DB_QUERY_FAIL"
	DB_NO_ROWS    = "DB_NO_ROWS"
	DB_CONN_FAIL  = "DB_CONN_FAIL"
	DB_TX_FAIL    = "DB_TX_FAIL"
	DB_CLOSE_FAIL = "DB_CLOSE_FAIL"
)

func (e *DBError) Error() string {
	return fmt.Sprintf(" %s [%s] | %v", e.Message, e.Code, e.Err)
}

func (e *DBError) Unwrap() error {
	return e.Err
}

func NewDBError(code, message string, err error) *DBError {
	return &DBError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
