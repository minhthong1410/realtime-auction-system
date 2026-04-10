package errors

import (
	"errors"
	"net/http"
)

var (
	ErrorTotpInvalidCode      = CustomError{Err: errors.New("totp.invalid_code"), StatusCode: http.StatusBadRequest}
	ErrorTotpNotEnabled       = CustomError{Err: errors.New("totp.not_enabled"), StatusCode: http.StatusBadRequest}
	ErrorTotpAlreadyEnabled   = CustomError{Err: errors.New("totp.already_enabled"), StatusCode: http.StatusConflict}
	ErrorTotpTempTokenExpired = CustomError{Err: errors.New("totp.temp_token_expired"), StatusCode: http.StatusUnauthorized}
	ErrorTotpTooManyAttempts  = CustomError{Err: errors.New("totp.too_many_attempts"), StatusCode: http.StatusTooManyRequests}
)
