package errors

import (
	"errors"
	"net/http"
)

type CustomError struct {
	Err        error
	StatusCode int
}

func (n CustomError) Error() string {
	return n.Err.Error()
}

func (n CustomError) ToHttpCode() int {
	return n.StatusCode
}

// Is allows errors.Is() to match against CustomError.
func (n CustomError) Is(target error) bool {
	t, ok := target.(CustomError)
	if !ok {
		return false
	}
	return n.Err.Error() == t.Err.Error()
}

var (
	ErrorInternalServer = CustomError{Err: errors.New("error.internal_server_error"), StatusCode: http.StatusInternalServerError}
	ErrorInvalidParams  = CustomError{Err: errors.New("error.invalid_params"), StatusCode: http.StatusBadRequest}
	ErrorUnauthorized   = CustomError{Err: errors.New("error.unauthorized"), StatusCode: http.StatusUnauthorized}
	ErrorNotFound       = CustomError{Err: errors.New("error.not_found"), StatusCode: http.StatusNotFound}
	ErrorDatabase       = CustomError{Err: errors.New("error.database_error"), StatusCode: http.StatusInternalServerError}
	ErrorExisted        = CustomError{Err: errors.New("error.existed"), StatusCode: http.StatusConflict}
)
