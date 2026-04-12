package errors

import (
	"errors"
	"net/http"
)

var (
	ErrorAuctionNotFound  = CustomError{Err: errors.New("auction.not_found"), StatusCode: http.StatusNotFound}
	ErrorAuctionEnded     = CustomError{Err: errors.New("auction.ended"), StatusCode: http.StatusBadRequest}
	ErrorInvalidEndTime   = CustomError{Err: errors.New("auction.invalid_end_time"), StatusCode: http.StatusBadRequest}
	ErrorBidTooLow        = CustomError{Err: errors.New("auction.bid_too_low"), StatusCode: http.StatusBadRequest}
	ErrorSelfBid          = CustomError{Err: errors.New("auction.self_bid"), StatusCode: http.StatusBadRequest}
	ErrorInsufficientFunds = CustomError{Err: errors.New("auction.insufficient_funds"), StatusCode: http.StatusBadRequest}
	ErrorNotAuctionOwner   = CustomError{Err: errors.New("auction.not_owner"), StatusCode: http.StatusForbidden}
	ErrorAuctionHasBids    = CustomError{Err: errors.New("auction.has_bids"), StatusCode: http.StatusBadRequest}
)
