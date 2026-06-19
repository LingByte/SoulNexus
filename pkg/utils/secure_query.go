package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SecureQueryBuilder secure query builder
type SecureQueryBuilder struct {
	Db *gorm.DB
}

// NewSecureQueryBuilder create a secure query builder
func NewSecureQueryBuilder(db *gorm.DB) *SecureQueryBuilder {
	return &SecureQueryBuilder{Db: db}
}

// ErrInvalidColumn is returned when a column name fails validation.
var ErrInvalidColumn = fmt.Errorf("invalid column name")

// ErrInvalidOperator is returned when an operator fails validation.
var ErrInvalidOperator = fmt.Errorf("invalid operator")

// ErrInvalidHavingCondition is returned when a HAVING condition fails validation.
var ErrInvalidHavingCondition = fmt.Errorf("invalid HAVING condition")

// ErrInvalidQuery is returned when a raw query fails validation.
var ErrInvalidQuery = fmt.Errorf("invalid query")

// ErrBetweenRequiresTwoValues is returned when BETWEEN receives != 2 values.
var ErrBetweenRequiresTwoValues = fmt.Errorf("BETWEEN operator requires exactly 2 values")

// ErrUnsupportedOperator is returned when an unsupported operator is used.
var ErrUnsupportedOperator = fmt.Errorf("unsupported operator")

// SafeWhere secure WHERE condition builder.
// It returns the chained *gorm.DB and a validation error.
// On validation failure the returned *gorm.DB is still valid (unchained) so callers
// can decide whether to abort or fall back.
func (sqb *SecureQueryBuilder) SafeWhere(column string, operator string, value interface{}) (*gorm.DB, error) {
	// Validate column name
	if !isValidColumnName(column) {
		return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidColumn, column)
	}

	// Validate operator
	if !isValidOperator(operator) {
		return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidOperator, operator)
	}

	// Build query based on operator
	switch strings.ToUpper(operator) {
	case "=", "!=", "<", ">", "<=", ">=":
		return sqb.Db.Where(fmt.Sprintf("%s %s ?", column, operator), value), nil
	case "<>":
		return sqb.Db.Where(fmt.Sprintf("%s <> ?", column), value), nil
	case "LIKE":
		if str, ok := value.(string); ok {
			return sqb.Db.Where(fmt.Sprintf("%s LIKE ?", column), "%"+str+"%"), nil
		}
		return sqb.Db.Where(fmt.Sprintf("%s LIKE ?", column), value), nil
	case "NOT LIKE":
		if str, ok := value.(string); ok {
			return sqb.Db.Where(fmt.Sprintf("%s NOT LIKE ?", column), "%"+str+"%"), nil
		}
		return sqb.Db.Where(fmt.Sprintf("%s NOT LIKE ?", column), value), nil
	case "IN":
		return sqb.Db.Where(fmt.Sprintf("%s IN ?", column), value), nil
	case "NOT IN":
		return sqb.Db.Where(fmt.Sprintf("%s NOT IN ?", column), value), nil
	case "BETWEEN":
		if values, ok := value.([]interface{}); ok && len(values) == 2 {
			return sqb.Db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", column), values[0], values[1]), nil
		}
		return sqb.Db, ErrBetweenRequiresTwoValues
	case "IS NULL":
		return sqb.Db.Where(fmt.Sprintf("%s IS NULL", column)), nil
	case "IS NOT NULL":
		return sqb.Db.Where(fmt.Sprintf("%s IS NOT NULL", column)), nil
	default:
		return sqb.Db, fmt.Errorf("%w: %s", ErrUnsupportedOperator, operator)
	}
}

// SafeOrder secure ORDER BY builder
func (sqb *SecureQueryBuilder) SafeOrder(column string, direction string) (*gorm.DB, error) {
	// Validate column name
	if !isValidColumnName(column) {
		return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidColumn, column)
	}

	// Validate sort direction
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC" // Default ascending
	}

	return sqb.Db.Order(fmt.Sprintf("%s %s", column, direction)), nil
}

// SafeSelect secure SELECT field builder
func (sqb *SecureQueryBuilder) SafeSelect(columns []string) (*gorm.DB, error) {
	// Validate all column names
	for _, column := range columns {
		if !isValidColumnName(column) {
			return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidColumn, column)
		}
	}

	return sqb.Db.Select(columns), nil
}

// SafeGroup secure GROUP BY builder
func (sqb *SecureQueryBuilder) SafeGroup(columns []string) (*gorm.DB, error) {
	// Validate all column names
	for _, column := range columns {
		if !isValidColumnName(column) {
			return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidColumn, column)
		}
	}

	return sqb.Db.Group(strings.Join(columns, ", ")), nil
}

// SafeHaving secure HAVING condition builder
func (sqb *SecureQueryBuilder) SafeHaving(condition string, args ...interface{}) (*gorm.DB, error) {
	// Validate column names in HAVING condition
	if !isValidHavingCondition(condition) {
		return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidHavingCondition, condition)
	}

	return sqb.Db.Having(condition, args...), nil
}

// isValidColumnName validate if column name is secure
func isValidColumnName(column string) bool {
	// Column names can only contain letters, numbers, underscores and dots
	pattern := `^[a-zA-Z_][a-zA-Z0-9_.]*$`
	matched, _ := regexp.MatchString(pattern, column)
	return matched && len(column) <= 64
}

// isValidOperator validate if operator is secure
func isValidOperator(operator string) bool {
	validOperators := map[string]bool{
		"=":           true,
		"!=":          true,
		"<>":          true,
		"<":           true,
		">":           true,
		"<=":          true,
		">=":          true,
		"LIKE":        true,
		"NOT LIKE":    true,
		"IN":          true,
		"NOT IN":      true,
		"BETWEEN":     true,
		"IS NULL":     true,
		"IS NOT NULL": true,
	}

	return validOperators[strings.ToUpper(operator)]
}

// isValidHavingCondition validate if HAVING condition is secure
func isValidHavingCondition(condition string) bool {
	dangerousKeywords := []string{
		"DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER",
		"EXEC", "EXECUTE", "UNION", "SCRIPT", "JAVASCRIPT",
	}
	upper := strings.ToUpper(condition)
	for _, kw := range dangerousKeywords {
		if strings.Contains(upper, kw) {
			return false
		}
	}
	// Allow parameter placeholders ?, percent sign %, single and double quotes, and common arithmetic symbols
	pattern := `^[a-zA-Z0-9_.,()\s=<>!%?\+\-\*/'"]+$`
	matched, _ := regexp.MatchString(pattern, condition)
	return matched
}

// SafeQuery execute secure raw query
func (sqb *SecureQueryBuilder) SafeQuery(query string, args ...interface{}) (*gorm.DB, error) {
	// Validate query statement
	if !isValidQuery(query) {
		return sqb.Db, fmt.Errorf("%w: %s", ErrInvalidQuery, query)
	}

	return sqb.Db.Raw(query, args...), nil
}

// isValidQuery validate if query statement is secure
func isValidQuery(query string) bool {
	// Convert to uppercase for checking
	upperQuery := strings.ToUpper(query)

	// Check for dangerous keywords
	dangerousKeywords := []string{
		"DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER",
		"EXEC", "EXECUTE", "UNION", "SCRIPT", "JAVASCRIPT",
		"TRUNCATE", "GRANT", "REVOKE", "SHUTDOWN",
	}

	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperQuery, keyword) {
			return false
		}
	}

	// Check if it starts with SELECT (only allow query operations)
	if !strings.HasPrefix(strings.TrimSpace(upperQuery), "SELECT") {
		return false
	}

	return true
}

// SafeTransaction execute secure transaction
func (sqb *SecureQueryBuilder) SafeTransaction(fn func(*gorm.DB) error) error {
	return sqb.Db.Transaction(func(tx *gorm.DB) error {
		// Create new transaction query builder
		txBuilder := NewSecureQueryBuilder(tx)

		// Execute transaction function
		return fn(txBuilder.Db)
	})
}

// SanitizeValue sanitize value to prevent injection
func SanitizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// Remove dangerous characters
		v = strings.ReplaceAll(v, "'", "''")
		v = strings.ReplaceAll(v, "\"", "\\\"")
		v = strings.ReplaceAll(v, "\\", "\\\\")
		return v
	case []string:
		// Sanitize string array
		sanitized := make([]string, len(v))
		for i, s := range v {
			sanitized[i] = SanitizeValue(s).(string)
		}
		return sanitized
	case time.Time:
		// Return time type directly
		return v
	case int, int8, int16, int32, int64:
		// Return integer type directly
		return v
	case uint, uint8, uint16, uint32, uint64:
		// Return unsigned integer type directly
		return v
	case float32, float64:
		// Return floating point type directly
		return v
	case bool:
		// Return boolean type directly
		return v
	default:
		// Convert other types to string and sanitize
		return SanitizeValue(fmt.Sprintf("%v", v))
	}
}

// ValidateInput validate input parameters
func ValidateInput(input interface{}) error {
	if input == nil {
		return nil
	}
	s := fmt.Sprintf("%v", input)
	if len(s) > 10000 {
		return fmt.Errorf("input too long")
	}
	sqlPatterns := []string{
		`(?i)\bunion\s+select\b`,
		`(?i)\bdrop\s+table\b`,
		`(?i)\bdelete\s+from\b`,
		`(?i)\binsert\s+into\s+\S+`,  // insert into <tbl>
		`(?i)\bupdate\s+\S+\s+set\b`, // update <tbl> set
		`(?i)\bor\s+1\s*=\s*1\b`,
		`(?i)\band\s+1\s*=\s*1\b`,
		`(?i)\bexec\s*\(`,
	}
	for _, p := range sqlPatterns {
		if matched, _ := regexp.MatchString(p, s); matched {
			return fmt.Errorf("potentially malicious input detected")
		}
	}
	return nil
}

// SafePaginate secure pagination query
func (sqb *SecureQueryBuilder) SafePaginate(page, pageSize int) *gorm.DB {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20 // Default 20 items per page
	}

	offset := (page - 1) * pageSize
	return sqb.Db.Offset(offset).Limit(pageSize)
}

// SafeCount secure count query
func (sqb *SecureQueryBuilder) SafeCount(model interface{}) (int64, error) {
	var count int64
	err := sqb.Db.Model(model).Count(&count).Error
	return count, err
}

// SafeExists secure existence check
func (sqb *SecureQueryBuilder) SafeExists(model interface{}, conditions map[string]interface{}) (bool, error) {
	query := sqb.Db.Model(model)

	// Safely add conditions
	for column, value := range conditions {
		if !isValidColumnName(column) {
			return false, fmt.Errorf("invalid column name: %s", column)
		}
		query = query.Where(fmt.Sprintf("%s = ?", column), value)
	}

	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

// SafeFirst secure first record query
func (sqb *SecureQueryBuilder) SafeFirst(dest interface{}, conditions map[string]interface{}) error {
	query := sqb.Db.Model(dest)

	// Safely add conditions
	for column, value := range conditions {
		if !isValidColumnName(column) {
			return fmt.Errorf("invalid column name: %s", column)
		}
		query = query.Where(fmt.Sprintf("%s = ?", column), value)
	}

	return query.First(dest).Error
}

// SafeFind secure batch query
func (sqb *SecureQueryBuilder) SafeFind(dest interface{}, conditions map[string]interface{}) error {
	query := sqb.Db.Model(dest)

	// Safely add conditions
	for column, value := range conditions {
		if !isValidColumnName(column) {
			return fmt.Errorf("invalid column name: %s", column)
		}
		query = query.Where(fmt.Sprintf("%s = ?", column), value)
	}

	return query.Find(dest).Error
}
