package errors

import (
	"errors"
	"net/http"
)

var (
	ErrorUserAlreadyExists  = CustomError{Err: errors.New("auth.user_already_exists"), StatusCode: http.StatusConflict}
	ErrorInvalidCredentials = CustomError{Err: errors.New("auth.invalid_credentials"), StatusCode: http.StatusUnauthorized}
	ErrorInvalidToken       = CustomError{Err: errors.New("auth.invalid_token"), StatusCode: http.StatusUnauthorized}
)
