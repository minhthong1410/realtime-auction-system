package errors

import (
	"errors"
	"net/http"
)

var (
	ErrorWithdrawalMinAmount     = CustomError{Err: errors.New("wallet.withdrawal_min_amount"), StatusCode: http.StatusBadRequest}
	ErrorWithdrawalPendingExists = CustomError{Err: errors.New("wallet.withdrawal_pending_exists"), StatusCode: http.StatusConflict}
)
