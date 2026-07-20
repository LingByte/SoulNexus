package common

import (
	"gorm.io/gorm"
)

const (
	DefaultPageSize    = 20
	DefaultMaxPageSize = 500
	MaxPageSize100     = 100
	MaxPageSize200     = 200
)

// PageParams holds normalized page/size after clamping.
type PageParams struct {
	Page   int
	Size   int
	Offset int
}

// NormalizePageParams clamps page and size. maxSize <= 0 uses DefaultMaxPageSize.
func NormalizePageParams(page, size, maxSize int) PageParams {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = DefaultPageSize
	}
	if maxSize <= 0 {
		maxSize = DefaultMaxPageSize
	}
	if size > maxSize {
		size = maxSize
	}
	return PageParams{
		Page:   page,
		Size:   size,
		Offset: (page - 1) * size,
	}
}

// TotalPages returns ceil(total/size) for pagination metadata.
func TotalPages(total int64, size int) int {
	if size <= 0 {
		return 0
	}
	n := int(total) / size
	if int(total)%size != 0 {
		n++
	}
	return n
}

// PagePayload builds a standard paginated list response body.
func PagePayload(list any, total int64, page, size int) map[string]any {
	return map[string]any{
		"list":      list,
		"total":     total,
		"page":      page,
		"pageSize":  size,
		"size":      size,
		"totalPage": TotalPages(total, size),
	}
}

// FindPage counts on q, then fetches one page ordered by orderExpr.
// maxSize <= 0 uses DefaultMaxPageSize.
func FindPage[T any](q *gorm.DB, page, size int, orderExpr string, maxSize int) ([]T, int64, error) {
	pp := NormalizePageParams(page, size, maxSize)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []T
	if err := q.Order(orderExpr).Offset(pp.Offset).Limit(pp.Size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindPageQuery is like FindPage but allows custom ordering (multiple Order calls, etc.).
func FindPageQuery[T any](q *gorm.DB, page, size, maxSize int, apply func(*gorm.DB) *gorm.DB) ([]T, int64, error) {
	pp := NormalizePageParams(page, size, maxSize)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []T
	if apply == nil {
		apply = func(db *gorm.DB) *gorm.DB { return db }
	}
	if err := apply(q).Offset(pp.Offset).Limit(pp.Size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
