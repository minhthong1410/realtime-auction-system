package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type BaseRequest struct {
	Pagination *PaginationRequest `json:"pagination,omitempty"`
	Data       interface{}        `json:"data,omitempty"`
}

func ParsePaginationFromQuery(r *http.Request) PaginationRequest {
	pagination := NewPaginationRequest()

	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			pagination.Page = pageNum
		}
	}

	if pageSize := r.URL.Query().Get("size"); pageSize != "" {
		if pageSizeNum, err := strconv.Atoi(pageSize); err == nil {
			pagination.Size = pageSizeNum
		}
	}

	if sort := r.URL.Query().Get("sortColumn"); sort != "" {
		pagination.SortColumn = sort
	}
	if direction := r.URL.Query().Get("sortDirection"); direction != "" {
		pagination.SortDirection = direction
	}

	pagination.Validate()
	return pagination
}

func ParseJSONRequest(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(r.Body)

	return decoder.Decode(dest)
}

func NewBaseRequest() *BaseRequest {
	return &BaseRequest{
		Pagination: &PaginationRequest{
			Page: 1,
			Size: 10,
		},
	}
}

func (b *BaseRequest) ValidateRequest() {
	if b.Pagination == nil {
		b.Pagination = &PaginationRequest{
			Page: 1,
			Size: 10,
		}
	} else {
		b.Pagination.Validate()
	}
}
