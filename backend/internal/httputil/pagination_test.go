package httputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPaginationRequestDefaults(t *testing.T) {
	p := NewPaginationRequest()
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 10, p.Size)
	assert.Equal(t, "created_at", p.SortColumn)
	assert.Equal(t, "desc", p.SortDirection)
}

func TestPaginationGetOffset(t *testing.T) {
	tests := []struct {
		page, size, expected int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 10, 20},
		{1, 20, 0},
		{5, 5, 20},
	}

	for _, tt := range tests {
		p := PaginationRequest{Page: tt.page, Size: tt.size}
		assert.Equal(t, tt.expected, p.GetOffset(), "page=%d size=%d", tt.page, tt.size)
	}
}

func TestPaginationGetLimit(t *testing.T) {
	p := PaginationRequest{Size: 25}
	assert.Equal(t, 25, p.GetLimit())
}

func TestPaginationValidate(t *testing.T) {
	tests := []struct {
		name      string
		input     PaginationRequest
		wantPage  int
		wantSize  int
		wantSort  string
		wantDir   string
	}{
		{"defaults on zero", PaginationRequest{}, 1, 10, "created_at", "desc"},
		{"negative page", PaginationRequest{Page: -1, Size: 10}, 1, 10, "created_at", "desc"},
		{"negative size", PaginationRequest{Page: 1, Size: -5}, 1, 10, "created_at", "desc"},
		{"size too large", PaginationRequest{Page: 1, Size: 10000}, 1, 100, "created_at", "desc"},
		{"size exactly 5000", PaginationRequest{Page: 1, Size: 5000}, 1, 5000, "created_at", "desc"},
		{"size 5001 clamped", PaginationRequest{Page: 1, Size: 5001}, 1, 100, "created_at", "desc"},
		{"valid custom", PaginationRequest{Page: 3, Size: 50, SortColumn: "amount", SortDirection: "asc"}, 3, 50, "amount", "asc"},
		{"invalid sort dir", PaginationRequest{Page: 1, Size: 10, SortDirection: "random"}, 1, 10, "created_at", "desc"},
		{"empty sort column", PaginationRequest{Page: 1, Size: 10, SortColumn: ""}, 1, 10, "created_at", "desc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.input
			p.Validate()
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantSize, p.Size)
			assert.Equal(t, tt.wantSort, p.SortColumn)
			assert.Equal(t, tt.wantDir, p.SortDirection)
		})
	}
}
