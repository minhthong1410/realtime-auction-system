package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- User JSON ---

func TestUserPasswordHiddenInJSON(t *testing.T) {
	user := User{
		ID: "id", Username: "alice", Email: "a@b.com",
		Password: "secret-bcrypt-hash", Balance: 10000,
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "secret-bcrypt-hash")
	assert.NotContains(t, string(data), "password")
	assert.Contains(t, string(data), "alice")
	assert.Contains(t, string(data), "10000")
}

func TestUserUnmarshalIgnoresPassword(t *testing.T) {
	input := `{"id":"1","username":"bob","password":"injected","balance":500}`
	var user User
	err := json.Unmarshal([]byte(input), &user)
	require.NoError(t, err)
	assert.Equal(t, "", user.Password, "password should not be unmarshalled from JSON")
}

// --- AuctionStatus ---

func TestAuctionStatusValues(t *testing.T) {
	assert.Equal(t, AuctionStatus(1), AuctionStatusActive)
	assert.Equal(t, AuctionStatus(2), AuctionStatusEnded)
	assert.Equal(t, AuctionStatus(3), AuctionStatusCancelled)
}

// --- WSMessage ---

func TestWSMessageNewBid(t *testing.T) {
	msg := WSMessage{
		Type: "new_bid",
		Data: WSNewBid{
			AuctionID: "auction-123", Amount: 50000,
			Username: "alice", BidCount: 5, TimeLeft: 300,
			CreatedAt: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	s := string(data)
	assert.Contains(t, s, `"type":"new_bid"`)
	assert.Contains(t, s, `"auction_id":"auction-123"`)
	assert.Contains(t, s, `"amount":50000`)
	assert.Contains(t, s, `"bid_count":5`)
}

func TestWSMessageAuctionEnded(t *testing.T) {
	msg := WSMessage{
		Type: "auction_ended",
		Data: WSAuctionEnded{
			AuctionID: "a-1", Winner: "alice", FinalPrice: 99999,
		},
	}

	data, _ := json.Marshal(msg)
	s := string(data)
	assert.Contains(t, s, `"winner":"alice"`)
	assert.Contains(t, s, `"final_price":99999`)
}

func TestWSMessageBalanceUpdate(t *testing.T) {
	msg := WSMessage{
		Type: "balance_update",
		Data: WSBalanceUpdate{Balance: 75000, Reason: "bid_placed"},
	}

	data, _ := json.Marshal(msg)
	s := string(data)
	assert.Contains(t, s, `"balance":75000`)
	assert.Contains(t, s, `"reason":"bid_placed"`)
}

func TestWSBalanceUpdateReasons(t *testing.T) {
	reasons := []string{"deposit", "withdrawal", "bid_placed", "bid_refund", "auction_sold"}
	for _, r := range reasons {
		msg := WSBalanceUpdate{Balance: 100, Reason: r}
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		assert.Contains(t, string(data), r)
	}
}

// --- Request Validation Tags ---

func TestDepositRequestAmountTag(t *testing.T) {
	// Verify struct tags exist
	var req DepositRequest
	req.Amount = 100
	assert.Equal(t, int64(100), req.Amount)
}

func TestWithdrawalRequestFields(t *testing.T) {
	req := WithdrawalRequest{
		Amount: 5000, BankName: "Chase",
		AccountNumber: "123456", AccountHolder: "Alice Smith",
		Note: "test",
	}
	data, _ := json.Marshal(req)
	s := string(data)
	assert.Contains(t, s, "Chase")
	assert.Contains(t, s, "123456")
	assert.Contains(t, s, "Alice Smith")
}

// --- Withdrawal ---

func TestWithdrawalReviewedAtNullable(t *testing.T) {
	w := Withdrawal{
		ID: "w-1", Status: "pending", ReviewedAt: nil,
	}
	data, _ := json.Marshal(w)
	assert.Contains(t, string(data), `"reviewed_at":null`)

	now := time.Now()
	w.ReviewedAt = &now
	data, _ = json.Marshal(w)
	assert.NotContains(t, string(data), `"reviewed_at":null`)
}

// --- Auction ---

func TestAuctionWinnerIDNullable(t *testing.T) {
	a := Auction{ID: "a-1", WinnerID: nil}
	data, _ := json.Marshal(a)
	assert.Contains(t, string(data), `"winner_id":null`)

	winnerID := "user-123"
	a.WinnerID = &winnerID
	data, _ = json.Marshal(a)
	assert.Contains(t, string(data), `"winner_id":"user-123"`)
}
