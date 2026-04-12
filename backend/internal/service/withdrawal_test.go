package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"9 digits", "123456789", "*****6789"},
		{"10 digits", "9876543210", "******3210"},
		{"5 chars", "12345", "*2345"},
		{"exactly 4", "1234", "1234"},
		{"3 chars", "123", "123"},
		{"2 chars", "12", "12"},
		{"1 char", "1", "1"},
		{"empty", "", ""},
		{"long account", "00112233445566778899", "****************8899"},
		{"letters mixed", "ABCD1234", "****1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, maskAccountNumber(tt.input))
		})
	}
}

func TestMaskAccountNumberOnlyShowsLast4(t *testing.T) {
	result := maskAccountNumber("9876543210")
	// First 6 should be masked
	for i := 0; i < 6; i++ {
		assert.Equal(t, byte('*'), result[i], "position %d should be masked", i)
	}
	// Last 4 should be visible
	assert.Equal(t, "3210", result[6:])
}

func TestMinWithdrawalAmount(t *testing.T) {
	assert.Equal(t, int64(500), minWithdrawalAmount)
}
