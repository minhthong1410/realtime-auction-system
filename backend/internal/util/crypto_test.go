package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testKey = []byte("01234567890123456789012345678901") // 32 bytes for AES-256

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"TOTP secret", "JBSWY3DPEHPK3PXP"},
		{"empty string", ""},
		{"single char", "a"},
		{"unicode", "unicode: こんにちは 🎉"},
		{"long string", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
		{"special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"newlines", "line1\nline2\rline3\r\n"},
		{"null bytes", "before\x00after"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, testKey)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			if tt.plaintext != "" {
				assert.NotEqual(t, tt.plaintext, encrypted)
			}

			decrypted, err := Decrypt(encrypted, testKey)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	// Same plaintext with same key → different ciphertexts (random nonce)
	results := make(map[string]bool)
	for i := 0; i < 20; i++ {
		enc, err := Encrypt("same text", testKey)
		require.NoError(t, err)
		assert.False(t, results[enc], "duplicate ciphertext at iteration %d", i)
		results[enc] = true
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	encrypted, err := Encrypt("secret data", testKey)
	require.NoError(t, err)

	wrongKey := []byte("99999999999999999999999999999999")
	_, err = Decrypt(encrypted, wrongKey)
	assert.Error(t, err)
}

func TestDecryptInvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"too short for nonce", "YQ=="},
		{"empty string", ""},
		{"valid base64 but corrupted", "SGVsbG8gV29ybGQ="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.input, testKey)
			assert.Error(t, err)
		})
	}
}

func TestEncryptInvalidKeyLengths(t *testing.T) {
	invalidKeys := [][]byte{
		[]byte("short"),
		[]byte(""),
		[]byte("16-byte-key!!!!"), // 15 bytes
		nil,
	}

	for _, key := range invalidKeys {
		_, err := Encrypt("test", key)
		assert.Error(t, err, "key length %d should fail", len(key))
	}
}

func TestEncryptValidKeyLengths(t *testing.T) {
	// AES supports 16, 24, 32 byte keys
	keys := [][]byte{
		[]byte("1234567890123456"),                 // 16 bytes (AES-128)
		[]byte("123456789012345678901234"),         // 24 bytes (AES-192)
		[]byte("12345678901234567890123456789012"), // 32 bytes (AES-256)
	}

	for _, key := range keys {
		enc, err := Encrypt("test", key)
		require.NoError(t, err, "key length %d should work", len(key))

		dec, err := Decrypt(enc, key)
		require.NoError(t, err)
		assert.Equal(t, "test", dec)
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	encrypted, err := Encrypt("secret", testKey)
	require.NoError(t, err)

	// Flip a byte in the middle
	tampered := []byte(encrypted)
	tampered[len(tampered)/2] ^= 0xFF
	_, err = Decrypt(string(tampered), testKey)
	assert.Error(t, err)
}
