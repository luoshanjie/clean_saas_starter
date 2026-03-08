package handler

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

type ListPageRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func paginateSlice[T any](items []T, page, pageSize int) ([]T, int, int, int) {
	page, pageSize = normalizePage(page, pageSize)
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, page, pageSize, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], page, pageSize, total
}
