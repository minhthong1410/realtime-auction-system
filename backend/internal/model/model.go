package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type AuctionStatus int16

const (
	AuctionStatusActive    AuctionStatus = 1
	AuctionStatusEnded     AuctionStatus = 2
	AuctionStatusCancelled AuctionStatus = 3
)

type Auction struct {
	ID            string        `json:"id"`
	SellerID      string        `json:"seller_id"`
	SellerName    string        `json:"seller_name,omitempty"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	ImageURL      string        `json:"image_url"`
	StartingPrice int64         `json:"starting_price"`
	CurrentPrice  int64         `json:"current_price"`
	WinnerID      *string       `json:"winner_id"`
	WinnerName    string        `json:"winner_name,omitempty"`
	Status        AuctionStatus `json:"status"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	CreatedAt     time.Time     `json:"created_at"`
	BidCount      int           `json:"bid_count,omitempty"`
}

type Bid struct {
	ID        string    `json:"id"`
	AuctionID string    `json:"auction_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// Request/Response types

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type LoginResponse struct {
	RequireTotpSetup bool   `json:"require_totp_setup"`
	TotpEnabled      bool   `json:"totp_enabled"`
	TempToken        string `json:"temp_token"`
}

type TotpSetupResponse struct {
	QRCode string `json:"qr_code"`
	Secret string `json:"secret"`
}

type TotpConfirmResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         User     `json:"user"`
	BackupCodes  []string `json:"backup_codes"`
}

type VerifyOtpRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

type TotpSetupRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
}

type TotpConfirmRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

type DepositRequest struct {
	Amount int64 `json:"amount" binding:"required,gt=0,max=10000000"` // max $100,000
}

type DepositResponse struct {
	CheckoutURL     string `json:"checkout_url"`
	StripePaymentID string `json:"stripe_payment_id"`
	Amount          int64  `json:"amount"`
}

type Deposit struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username,omitempty"`
	Amount          int64     `json:"amount"`
	Status          string    `json:"status"`
	StripePaymentID string    `json:"stripe_payment_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WithdrawalRequest struct {
	Amount        int64  `json:"amount" binding:"required,gt=0,max=10000000"` // max $100,000
	BankName      string `json:"bank_name" binding:"required,max=100"`
	AccountNumber string `json:"account_number" binding:"required,max=50"`
	AccountHolder string `json:"account_holder" binding:"required,max=100"`
	Note          string `json:"note" binding:"max=255"`
}

type Withdrawal struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Username      string     `json:"username,omitempty"`
	Amount        int64      `json:"amount"`
	Status        string     `json:"status"`
	BankName      string     `json:"bank_name"`
	AccountNumber string     `json:"account_number"`
	AccountHolder string     `json:"account_holder"`
	Note          string     `json:"note"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateAuctionRequest struct {
	Title         string    `json:"title" binding:"required,max=255"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	StartingPrice int64     `json:"starting_price" binding:"required,gt=0"`
	EndTime       time.Time `json:"end_time" binding:"required"`
}

type PlaceBidRequest struct {
	Amount int64 `json:"amount" binding:"required,gt=0"`
}

// WebSocket message types

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type WSNewBid struct {
	AuctionID string    `json:"auction_id"`
	Amount    int64     `json:"amount"`
	Username  string    `json:"username"`
	BidCount  int       `json:"bid_count"`
	TimeLeft  int64     `json:"time_left"`
	CreatedAt time.Time `json:"created_at"`
}

type WSAuctionEnded struct {
	AuctionID  string `json:"auction_id"`
	Winner     string `json:"winner"`
	FinalPrice int64  `json:"final_price"`
}

type WSBalanceUpdate struct {
	Balance int64  `json:"balance"`
	Reason  string `json:"reason"`
}
