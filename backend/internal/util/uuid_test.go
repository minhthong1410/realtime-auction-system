package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUUID(t *testing.T) {
	id := NewUUID()
	assert.Len(t, id, 16)

	// Uniqueness: 100 UUIDs should all be different
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := UUIDToString(NewUUID())
		assert.False(t, seen[s], "duplicate UUID generated: %s", s)
		seen[s] = true
	}
}

func TestUUIDToString(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect string
	}{
		{"valid 16 bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}, "01020304-0506-0708-090a-0b0c0d0e0f10"},
		{"all zeros", []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, "00000000-0000-0000-0000-000000000000"},
		{"all ff", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, "ffffffff-ffff-ffff-ffff-ffffffffffff"},
		{"empty", []byte{}, ""},
		{"nil", nil, ""},
		{"too short (1 byte)", []byte{0x01}, ""},
		{"too short (15 bytes)", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, ""},
		{"too long (17 bytes)", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, UUIDToString(tt.input))
		})
	}
}

func TestUUIDFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid standard", "01020304-0506-0708-090a-0b0c0d0e0f10", false},
		{"valid v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid nil UUID", "00000000-0000-0000-0000-000000000000", false},
		{"uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"invalid format", "not-a-uuid", true},
		{"empty string", "", true},
		{"too short", "550e8400", true},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false}, // uuid.Parse accepts this
		{"has braces", "{550e8400-e29b-41d4-a716-446655440000}", false}, // uuid.Parse accepts braces
		{"has spaces", " 550e8400-e29b-41d4-a716-446655440000 ", false}, // uuid.Parse trims spaces
		{"invalid hex char", "550g8400-e29b-41d4-a716-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UUIDFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, 16)
			}
		})
	}
}

func TestUUIDRoundTrip(t *testing.T) {
	for i := 0; i < 50; i++ {
		original := NewUUID()
		str := UUIDToString(original)
		assert.NotEmpty(t, str)
		assert.Len(t, str, 36) // UUID string is always 36 chars

		parsed, err := UUIDFromString(str)
		require.NoError(t, err)
		assert.Equal(t, original, parsed)
	}
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.True(t, IsValidUUID("00000000-0000-0000-0000-000000000000"))
	assert.False(t, IsValidUUID("not-valid"))
	assert.False(t, IsValidUUID(""))
	assert.False(t, IsValidUUID("550e8400-e29b-41d4-a716")) // incomplete
}
