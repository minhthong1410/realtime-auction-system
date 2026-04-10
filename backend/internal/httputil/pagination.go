package httputil

type PaginationRequest struct {
	Page          int    `json:"page" form:"page"`
	Size          int    `json:"size" form:"size"`
	SortColumn    string `json:"sortColumn" form:"sortColumn"`
	SortDirection string `json:"sortDirection" form:"sortDirection"`
}

func NewPaginationRequest() PaginationRequest {
	return PaginationRequest{
		Page:          1,
		Size:          10,
		SortColumn:    "created_at",
		SortDirection: "desc",
	}
}

func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.Size
}

func (p *PaginationRequest) GetLimit() int {
	return p.Size
}

func (p *PaginationRequest) Validate() {
	if p.Page <= 0 {
		p.Page = 1
	}

	if p.Size <= 0 {
		p.Size = 10
	}

	if p.Size > 5000 {
		p.Size = 100
	}

	if p.SortColumn == "" {
		p.SortColumn = "created_at"
	}

	if p.SortDirection == "" || (p.SortDirection != "asc" && p.SortDirection != "desc") {
		p.SortDirection = "desc"
	}
}
