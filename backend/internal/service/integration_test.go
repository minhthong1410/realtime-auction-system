//go:build integration

package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kurama/auction-system/backend/internal/config"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testDB      *sql.DB
	testRDB     *redis.Client
	testQueries *repository.Queries
	testHub     *ws.Hub
)

func TestMain(m *testing.M) {
	l, _ := zap.NewDevelopment()
	logger.Init(l)

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(localhost:3306)/auction_test?parseTime=true&loc=UTC"
	}

	var err error
	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("skip integration tests: %v\n", err)
		os.Exit(0)
	}
	if err := testDB.Ping(); err != nil {
		fmt.Printf("skip integration tests: cannot connect to MySQL: %v\n", err)
		os.Exit(0)
	}

	testRDB = redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := testRDB.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("skip integration tests: cannot connect to Redis: %v\n", err)
		os.Exit(0)
	}

	testQueries = repository.New(testDB)
	testHub = ws.NewHub(testRDB)

	// Create tables if not exist
	setupTestDB()

	code := m.Run()

	testDB.Close()
	testRDB.Close()
	os.Exit(code)
}

func setupTestDB() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BINARY(16) NOT NULL PRIMARY KEY,
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			balance BIGINT NOT NULL DEFAULT 0,
			totp_secret VARCHAR(255) DEFAULT NULL,
			totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			backup_codes JSON DEFAULT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS auctions (
			id BINARY(16) NOT NULL PRIMARY KEY,
			seller_id BINARY(16) NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			image_url VARCHAR(512) DEFAULT '',
			starting_price BIGINT NOT NULL,
			current_price BIGINT NOT NULL,
			winner_id BINARY(16) DEFAULT NULL,
			status SMALLINT NOT NULL DEFAULT 1,
			start_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			end_time TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bids (
			id BINARY(16) NOT NULL PRIMARY KEY,
			auction_id BINARY(16) NOT NULL,
			user_id BINARY(16) NOT NULL,
			amount BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS deposits (
			id BINARY(16) NOT NULL PRIMARY KEY,
			user_id BINARY(16) NOT NULL,
			amount BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			stripe_payment_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id BINARY(16) NOT NULL PRIMARY KEY,
			user_id BINARY(16) NOT NULL,
			amount BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			bank_name VARCHAR(100) NOT NULL,
			account_number VARCHAR(50) NOT NULL,
			account_holder VARCHAR(100) NOT NULL,
			note VARCHAR(255) NOT NULL DEFAULT '',
			reviewed_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range statements {
		testDB.Exec(s)
	}
}

func cleanupUser(t *testing.T, id []byte) {
	t.Helper()
	testDB.Exec("DELETE FROM withdrawals WHERE user_id = ?", id)
	testDB.Exec("DELETE FROM deposits WHERE user_id = ?", id)
	testDB.Exec("DELETE FROM bids WHERE user_id = ?", id)
	testDB.Exec("DELETE FROM users WHERE id = ?", id)
}

func createTestUser(t *testing.T, username, email string, balance int64) []byte {
	t.Helper()
	id := util.NewUUID()
	err := testQueries.CreateUser(context.Background(), repository.CreateUserParams{
		ID: id, Username: username, Email: email, Password: "$2a$10$fakehash",
	})
	require.NoError(t, err)
	if balance > 0 {
		testDB.Exec("UPDATE users SET balance = ? WHERE id = ?", balance, id)
	}
	t.Cleanup(func() { cleanupUser(t, id) })
	return id
}

// ========== DEPOSIT TESTS ==========

func TestHandlePaymentSuccessIntegration(t *testing.T) {
	userID := createTestUser(t, "dep_user1", "dep1@test.com", 0)
	depositService := NewDepositService(testDB, testQueries, testHub, "http://localhost")
	ctx := context.Background()

	// Create deposit record
	depositID := util.NewUUID()
	err := testQueries.CreateDeposit(ctx, repository.CreateDepositParams{
		ID: depositID, UserID: userID, Amount: 10000, StripePaymentID: "cs_test_dep1",
	})
	require.NoError(t, err)

	// Process payment success
	err = depositService.HandlePaymentSuccess(ctx, "cs_test_dep1")
	require.NoError(t, err)

	// Verify balance increased
	balance, _ := testQueries.GetBalance(ctx, userID)
	assert.Equal(t, int64(10000), balance)
}

func TestHandlePaymentSuccessIdempotent(t *testing.T) {
	userID := createTestUser(t, "dep_user2", "dep2@test.com", 0)
	depositService := NewDepositService(testDB, testQueries, testHub, "http://localhost")
	ctx := context.Background()

	depositID := util.NewUUID()
	testQueries.CreateDeposit(ctx, repository.CreateDepositParams{
		ID: depositID, UserID: userID, Amount: 5000, StripePaymentID: "cs_test_dep2",
	})

	// Call twice — should be idempotent
	depositService.HandlePaymentSuccess(ctx, "cs_test_dep2")
	depositService.HandlePaymentSuccess(ctx, "cs_test_dep2")

	balance, _ := testQueries.GetBalance(ctx, userID)
	assert.Equal(t, int64(5000), balance, "should only credit once")
}

func TestHandlePaymentSuccessNonexistent(t *testing.T) {
	depositService := NewDepositService(testDB, testQueries, testHub, "http://localhost")
	err := depositService.HandlePaymentSuccess(context.Background(), "cs_nonexistent")
	assert.NoError(t, err, "should silently ignore unknown stripe ID")
}

func TestHandlePaymentFailed(t *testing.T) {
	userID := createTestUser(t, "dep_user3", "dep3@test.com", 0)
	depositService := NewDepositService(testDB, testQueries, testHub, "http://localhost")
	ctx := context.Background()

	depositID := util.NewUUID()
	testQueries.CreateDeposit(ctx, repository.CreateDepositParams{
		ID: depositID, UserID: userID, Amount: 3000, StripePaymentID: "cs_test_dep3",
	})

	err := depositService.HandlePaymentFailed(ctx, "cs_test_dep3")
	require.NoError(t, err)

	// Verify balance unchanged
	balance, _ := testQueries.GetBalance(ctx, userID)
	assert.Equal(t, int64(0), balance)
}

// ========== WITHDRAWAL TESTS ==========

func TestCreateWithdrawalSuccess(t *testing.T) {
	userID := createTestUser(t, "wd_user1", "wd1@test.com", 100000)
	withdrawalService := NewWithdrawalService(testDB, testQueries, testHub)
	ctx := context.Background()

	w, err := withdrawalService.CreateWithdrawal(ctx, util.UUIDToString(userID), model.WithdrawalRequest{
		Amount: 5000, BankName: "Chase", AccountNumber: "123456", AccountHolder: "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", w.Status)
	assert.Equal(t, int64(5000), w.Amount)

	// Balance should be deducted
	balance, _ := testQueries.GetBalance(ctx, userID)
	assert.Equal(t, int64(95000), balance)
}

func TestCreateWithdrawalInsufficientFunds(t *testing.T) {
	userID := createTestUser(t, "wd_user2", "wd2@test.com", 100)
	withdrawalService := NewWithdrawalService(testDB, testQueries, testHub)

	_, err := withdrawalService.CreateWithdrawal(context.Background(), util.UUIDToString(userID), model.WithdrawalRequest{
		Amount: 5000, BankName: "Chase", AccountNumber: "123", AccountHolder: "Bob",
	})
	assert.Error(t, err, "should reject: insufficient funds")

	// Balance unchanged
	balance, _ := testQueries.GetBalance(context.Background(), userID)
	assert.Equal(t, int64(100), balance)
}

func TestCreateWithdrawalBelowMinimum(t *testing.T) {
	userID := createTestUser(t, "wd_user3", "wd3@test.com", 100000)
	withdrawalService := NewWithdrawalService(testDB, testQueries, testHub)

	_, err := withdrawalService.CreateWithdrawal(context.Background(), util.UUIDToString(userID), model.WithdrawalRequest{
		Amount: 100, BankName: "Chase", AccountNumber: "123", AccountHolder: "Carol",
	})
	assert.Error(t, err, "should reject: below minimum")
}

func TestCreateWithdrawalDuplicatePending(t *testing.T) {
	userID := createTestUser(t, "wd_user4", "wd4@test.com", 100000)
	withdrawalService := NewWithdrawalService(testDB, testQueries, testHub)
	ctx := context.Background()

	// First withdrawal
	_, err := withdrawalService.CreateWithdrawal(ctx, util.UUIDToString(userID), model.WithdrawalRequest{
		Amount: 5000, BankName: "Chase", AccountNumber: "123", AccountHolder: "Dave",
	})
	require.NoError(t, err)

	// Second should fail
	_, err = withdrawalService.CreateWithdrawal(ctx, util.UUIDToString(userID), model.WithdrawalRequest{
		Amount: 5000, BankName: "Chase", AccountNumber: "456", AccountHolder: "Dave",
	})
	assert.Error(t, err, "should reject: pending withdrawal exists")
}

// ========== BID TESTS ==========

func TestPlaceBidSuccess(t *testing.T) {
	sellerID := createTestUser(t, "bid_seller1", "bseller1@test.com", 0)
	bidderID := createTestUser(t, "bid_bidder1", "bbidder1@test.com", 100000)
	ctx := context.Background()

	// Create auction
	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "Test Item",
		StartingPrice: 1000, CurrentPrice: 1000,
		EndTime: time.Now().Add(1 * time.Hour),
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM bids WHERE auction_id = ?", auctionID)
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)
	bid, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidderID), "bid_bidder1", 2000)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), bid.Amount)

	// Bidder balance deducted
	balance, _ := testQueries.GetBalance(ctx, bidderID)
	assert.Equal(t, int64(98000), balance)
}

func TestPlaceBidSelfBidRejected(t *testing.T) {
	sellerID := createTestUser(t, "bid_selfbid", "selfbid@test.com", 100000)
	ctx := context.Background()

	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "My Item",
		StartingPrice: 1000, CurrentPrice: 1000,
		EndTime: time.Now().Add(1 * time.Hour),
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)
	_, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(sellerID), "bid_selfbid", 2000)
	assert.Error(t, err, "should reject self-bid")
}

func TestPlaceBidTooLow(t *testing.T) {
	sellerID := createTestUser(t, "bid_low_seller", "blseller@test.com", 0)
	bidderID := createTestUser(t, "bid_low_bidder", "blbidder@test.com", 100000)
	ctx := context.Background()

	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "Low Bid Item",
		StartingPrice: 5000, CurrentPrice: 5000,
		EndTime: time.Now().Add(1 * time.Hour),
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)
	_, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidderID), "bid_low_bidder", 3000)
	assert.Error(t, err, "bid below current price should be rejected")
}

func TestPlaceBidInsufficientBalance(t *testing.T) {
	sellerID := createTestUser(t, "bid_poor_seller", "bpseller@test.com", 0)
	bidderID := createTestUser(t, "bid_poor_bidder", "bpbidder@test.com", 100) // only $1
	ctx := context.Background()

	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "Expensive Item",
		StartingPrice: 1000, CurrentPrice: 1000,
		EndTime: time.Now().Add(1 * time.Hour),
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)
	_, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidderID), "bid_poor_bidder", 2000)
	assert.Error(t, err, "should reject: insufficient funds")
}

func TestPlaceBidRefundsPreviousBidder(t *testing.T) {
	sellerID := createTestUser(t, "ref_seller", "refseller@test.com", 0)
	bidder1ID := createTestUser(t, "ref_bidder1", "refbidder1@test.com", 100000)
	bidder2ID := createTestUser(t, "ref_bidder2", "refbidder2@test.com", 100000)
	ctx := context.Background()

	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "Refund Test",
		StartingPrice: 1000, CurrentPrice: 1000,
		EndTime: time.Now().Add(1 * time.Hour),
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM bids WHERE auction_id = ?", auctionID)
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)

	// Bidder1 bids 2000
	_, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidder1ID), "ref_bidder1", 2000)
	require.NoError(t, err)

	b1Balance, _ := testQueries.GetBalance(ctx, bidder1ID)
	assert.Equal(t, int64(98000), b1Balance) // 100000 - 2000

	// Bidder2 bids higher → bidder1 gets refund
	_, err = bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidder2ID), "ref_bidder2", 3000)
	require.NoError(t, err)

	b1BalanceAfter, _ := testQueries.GetBalance(ctx, bidder1ID)
	assert.Equal(t, int64(100000), b1BalanceAfter, "bidder1 should be fully refunded")

	b2Balance, _ := testQueries.GetBalance(ctx, bidder2ID)
	assert.Equal(t, int64(97000), b2Balance, "bidder2 should have 3000 deducted")
}

func TestPlaceBidOnEndedAuction(t *testing.T) {
	sellerID := createTestUser(t, "ended_seller", "eseller@test.com", 0)
	bidderID := createTestUser(t, "ended_bidder", "ebidder@test.com", 100000)
	ctx := context.Background()

	auctionID := util.NewUUID()
	testQueries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID: auctionID, SellerID: sellerID, Title: "Ended Auction",
		StartingPrice: 1000, CurrentPrice: 1000,
		EndTime: time.Now().Add(-1 * time.Hour), // already ended
	})
	t.Cleanup(func() {
		testDB.Exec("DELETE FROM auctions WHERE id = ?", auctionID)
	})

	bidService := NewBidService(testDB, testQueries, testHub)
	_, err := bidService.PlaceBid(ctx,
		util.UUIDToString(auctionID), util.UUIDToString(bidderID), "ended_bidder", 2000)
	assert.Error(t, err, "should reject bid on ended auction")
}

// ========== AUTH INTEGRATION ==========

func TestRegisterAndLogin(t *testing.T) {
	authService := NewAuthService(testDB, testQueries, config.JWTConfig{
		Secret: "test-secret", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour,
	})
	ctx := context.Background()

	// Register
	user, err := authService.Register(ctx, model.RegisterRequest{
		Username: "integ_user1", Email: "integ1@test.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "integ_user1", user.Username)
	t.Cleanup(func() {
		id, _ := util.UUIDFromString(user.ID)
		cleanupUser(t, id)
	})

	// Login with correct creds
	loggedIn, err := authService.ValidateCredentials(ctx, model.LoginRequest{
		Email: "integ1@test.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, user.ID, loggedIn.ID)

	// Login with wrong password
	_, err = authService.ValidateCredentials(ctx, model.LoginRequest{
		Email: "integ1@test.com", Password: "wrong",
	})
	assert.Error(t, err)

	// Login with nonexistent email
	_, err = authService.ValidateCredentials(ctx, model.LoginRequest{
		Email: "nobody@test.com", Password: "password123",
	})
	assert.Error(t, err)
}

func TestRegisterDuplicate(t *testing.T) {
	authService := NewAuthService(testDB, testQueries, config.JWTConfig{Secret: "s"})
	ctx := context.Background()

	user, _ := authService.Register(ctx, model.RegisterRequest{
		Username: "dup_user", Email: "dup@test.com", Password: "pass",
	})
	t.Cleanup(func() {
		id, _ := util.UUIDFromString(user.ID)
		cleanupUser(t, id)
	})

	// Same email
	_, err := authService.Register(ctx, model.RegisterRequest{
		Username: "dup_user2", Email: "dup@test.com", Password: "pass",
	})
	assert.Error(t, err)

	// Same username
	_, err = authService.Register(ctx, model.RegisterRequest{
		Username: "dup_user", Email: "dup2@test.com", Password: "pass",
	})
	assert.Error(t, err)
}
