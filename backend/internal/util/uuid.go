package util

import (
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// NewUUID generates a new UUID v4 and returns it as BINARY(16) bytes.
func NewUUID() []byte {
	id := uuid.New()
	return id[:]
}

// UUIDToString converts BINARY(16) bytes to a UUID string.
func UUIDToString(b []byte) string {
	if len(b) != 16 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// UUIDFromString converts a UUID string to BINARY(16) bytes.
func UUIDFromString(s string) ([]byte, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	b := id[:]
	return b, nil
}

// IsValidUUID checks if a string is a valid UUID.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
