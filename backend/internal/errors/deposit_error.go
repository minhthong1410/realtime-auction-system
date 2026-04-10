package errors

import (
	"errors"
	"net/http"
)

var (
	ErrorDepositNotFound   = CustomError{Err: errors.New("deposit.not_found"), StatusCode: http.StatusNotFound}
	ErrorDepositNotPending = CustomError{Err: errors.New("deposit.not_pending"), StatusCode: http.StatusBadRequest}
)
