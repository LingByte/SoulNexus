package validator

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/i18n"
)

// Validator validates data
type Validator struct {
	rules       map[string][]Rule
	customRules map[string]RuleFunc
	i18n        *i18n.Manager
	mu          sync.RWMutex
}

// Rule represents a validation rule
type Rule struct {
	Name    string
	Func    RuleFunc
	Message string
	Params  map[string]interface{}
}

// RuleFunc is a validation function
type RuleFunc func(value interface{}, params map[string]interface{}) error

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Rule    string
	Message string
	Value   interface{}
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	messages := make([]string, len(e))
	for i, err := range e {
		messages[i] = err.Message
	}
	return strings.Join(messages, "; ")
}

// NewValidator creates a new validator
func NewValidator(i18nManager *i18n.Manager) *Validator {
	v := &Validator{
		rules:       make(map[string][]Rule),
		customRules: make(map[string]RuleFunc),
		i18n:        i18nManager,
	}

	// Register default rules
	v.registerDefaultRules()

	return v
}

// RegisterRule registers a validation rule for a field
func (v *Validator) RegisterRule(field string, rule Rule) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.rules[field] == nil {
		v.rules[field] = make([]Rule, 0)
	}
	v.rules[field] = append(v.rules[field], rule)
}

// RegisterCustomRule registers a custom rule function
func (v *Validator) RegisterCustomRule(name string, ruleFunc RuleFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customRules[name] = ruleFunc
}

// Validate validates a struct
func (v *Validator) Validate(data interface{}, locale i18n.Locale) ValidationErrors {
	var errors ValidationErrors

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ValidationErrors{
			&ValidationError{
				Field:   "",
				Rule:    "type",
				Message: "data must be a struct",
			},
		}
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Get field name from tag or use struct field name
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Tag.Get("form")
		}
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}
		if fieldName == "-" {
			continue
		}

		// Get validation rules from tag
		validateTag := field.Tag.Get("validate")
		if validateTag != "" {
			rules := v.parseTagRules(validateTag)
			for _, rule := range rules {
				if err := v.validateField(fieldName, fieldValue.Interface(), rule, locale); err != nil {
					errors = append(errors, err)
				}
			}
		}

		// Apply registered rules
		if rules, ok := v.rules[fieldName]; ok {
			for _, rule := range rules {
				if err := v.validateField(fieldName, fieldValue.Interface(), rule, locale); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	return errors
}

// ValidateField validates a single field
func (v *Validator) ValidateField(field string, value interface{}, rules []Rule, locale i18n.Locale) ValidationErrors {
	var errors ValidationErrors

	for _, rule := range rules {
		if err := v.validateField(field, value, rule, locale); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// validateField validates a field with a rule
func (v *Validator) validateField(field string, value interface{}, rule Rule, locale i18n.Locale) *ValidationError {
	// Check custom rules first
	if ruleFunc, ok := v.customRules[rule.Name]; ok {
		if err := ruleFunc(value, rule.Params); err != nil {
			message := v.getErrorMessage(field, rule, locale, value)
			return &ValidationError{
				Field:   field,
				Rule:    rule.Name,
				Message: message,
				Value:   value,
			}
		}
		return nil
	}

	// Use rule function
	if rule.Func != nil {
		if err := rule.Func(value, rule.Params); err != nil {
			message := v.getErrorMessage(field, rule, locale, value)
			return &ValidationError{
				Field:   field,
				Rule:    rule.Name,
				Message: message,
				Value:   value,
			}
		}
	}

	return nil
}

// getErrorMessage gets localized error message
func (v *Validator) getErrorMessage(field string, rule Rule, locale i18n.Locale, value interface{}) string {
	// Try custom message first
	if rule.Message != "" {
		return rule.Message
	}

	// Try i18n
	if v.i18n != nil {
		key := fmt.Sprintf("validation.%s.%s", field, rule.Name)
		message := v.i18n.GetTranslation(locale, key)
		if message != key {
			return fmt.Sprintf(message, value)
		}

		// Try generic rule message
		key = fmt.Sprintf("validation.rule.%s", rule.Name)
		message = v.i18n.GetTranslation(locale, key)
		if message != key {
			return fmt.Sprintf(message, field, value)
		}
	}

	// Default message
	return fmt.Sprintf("validation failed for field '%s' with rule '%s'", field, rule.Name)
}

// parseTagRules parses validation tag
func (v *Validator) parseTagRules(tag string) []Rule {
	var rules []Rule

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse rule name and params (support both ":" and "=")
		var ruleName string
		var paramValue string
		params := make(map[string]interface{})

		if idx := strings.Index(part, ":"); idx != -1 {
			ruleName = strings.TrimSpace(part[:idx])
			paramValue = strings.TrimSpace(part[idx+1:])
		} else if idx := strings.Index(part, "="); idx != -1 {
			ruleName = strings.TrimSpace(part[:idx])
			paramValue = strings.TrimSpace(part[idx+1:])
		} else {
			ruleName = strings.TrimSpace(part)
		}

		if paramValue != "" {
			params["value"] = paramValue
		}

		// Get rule function
		if ruleFunc, ok := v.customRules[ruleName]; ok {
			rules = append(rules, Rule{
				Name:   ruleName,
				Func:   ruleFunc,
				Params: params,
			})
		} else if defaultRule, ok := defaultRules[ruleName]; ok {
			rules = append(rules, Rule{
				Name:   ruleName,
				Func:   defaultRule,
				Params: params,
			})
		}
	}

	return rules
}

// registerDefaultRules registers default validation rules
func (v *Validator) registerDefaultRules() {
	// Register all default rules
	for name, ruleFunc := range defaultRules {
		v.customRules[name] = ruleFunc
	}
}

// defaultRules contains built-in validation rules
var defaultRules = map[string]RuleFunc{
	"required": validateRequired,
	"email":    validateEmail,
	"xss":      validateXSS,
	"sql":      validateSQL,
	"min":      validateMin,
	"max":      validateMax,
	"minlen":   validateMinLen,
	"maxlen":   validateMaxLen,
	"url":      validateURL,
	"phone":    validatePhone,
	"alphanum": validateAlphanum,
	"numeric":  validateNumeric,
	"alpha":    validateAlpha,
}

// validateRequired checks if value is not empty
func validateRequired(value interface{}, params map[string]interface{}) error {
	if value == nil {
		return fmt.Errorf("required")
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("required")
		}
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("required")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("required")
		}
	}

	return nil
}

// validateEmail validates email format
func validateEmail(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("email must be a string")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// validateXSS checks for XSS attacks
func validateXSS(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // Not a string, skip XSS check
	}

	// Common XSS patterns
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)data:text/html`),
		regexp.MustCompile(`(?i)vbscript:`),
		regexp.MustCompile(`(?i)expression\s*\(`),
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(str) {
			return fmt.Errorf("potential XSS attack detected")
		}
	}

	return nil
}

// validateSQL checks for SQL injection
func validateSQL(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // Not a string, skip SQL check
	}

	// Common SQL injection patterns
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\bunion\b.*\bselect\b)`),
		regexp.MustCompile(`(?i)(\bselect\b.*\bfrom\b)`),
		regexp.MustCompile(`(?i)(\binsert\b.*\binto\b)`),
		regexp.MustCompile(`(?i)(\bupdate\b.*\bset\b)`),
		regexp.MustCompile(`(?i)(\bdelete\b.*\bfrom\b)`),
		regexp.MustCompile(`(?i)(\bdrop\b.*\btable\b)`),
		regexp.MustCompile(`(?i)(\bexec\b|\bexecute\b)`),
		regexp.MustCompile(`(?i)(\bexecutive\b)`),
		regexp.MustCompile(`(?i)(\bor\b.*['"]?\s*1\s*=\s*1\b)`),
		regexp.MustCompile(`(?i)(\bor\b.*['"]1['"]\s*=\s*['"]1['"])`),
		regexp.MustCompile(`(?i)(\band\b.*['"]?\s*1\s*=\s*1\b)`),
		regexp.MustCompile(`(?i)(--|\/\*|\*\/)`),
		regexp.MustCompile(`(?i)(\bxp_\w+\b)`),
		regexp.MustCompile(`(?i)(\bsp_\w+\b)`),
		regexp.MustCompile(`(?i)(\bor\b\s+['"]1['"]\s*=\s*['"]1['"])`),
	}

	for _, pattern := range sqlPatterns {
		if pattern.MatchString(str) {
			return fmt.Errorf("potential SQL injection detected")
		}
	}

	return nil
}

// validateMin validates minimum value
func validateMin(value interface{}, params map[string]interface{}) error {
	minVal, ok := params["value"]
	if !ok {
		return fmt.Errorf("min rule requires a value parameter")
	}

	minFloat, err := toFloat64(minVal)
	if err != nil {
		return fmt.Errorf("invalid min value: %v", minVal)
	}

	valFloat, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value must be numeric")
	}

	if valFloat < minFloat {
		return fmt.Errorf("value must be at least %v", minFloat)
	}

	return nil
}

// validateMax validates maximum value
func validateMax(value interface{}, params map[string]interface{}) error {
	maxVal, ok := params["value"]
	if !ok {
		return fmt.Errorf("max rule requires a value parameter")
	}

	maxFloat, err := toFloat64(maxVal)
	if err != nil {
		return fmt.Errorf("invalid max value: %v", maxVal)
	}

	valFloat, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value must be numeric")
	}

	if valFloat > maxFloat {
		return fmt.Errorf("value must be at most %v", maxFloat)
	}

	return nil
}

// validateMinLen validates minimum length
func validateMinLen(value interface{}, params map[string]interface{}) error {
	minLen, ok := params["value"]
	if !ok {
		return fmt.Errorf("minlen rule requires a value parameter")
	}

	minInt, err := toInt(minLen)
	if err != nil {
		return fmt.Errorf("invalid minlen value: %v", minLen)
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("minlen applies to strings only")
	}

	if len(str) < minInt {
		return fmt.Errorf("length must be at least %d", minInt)
	}

	return nil
}

// validateMaxLen validates maximum length
func validateMaxLen(value interface{}, params map[string]interface{}) error {
	maxLen, ok := params["value"]
	if !ok {
		return fmt.Errorf("maxlen rule requires a value parameter")
	}

	maxInt, err := toInt(maxLen)
	if err != nil {
		return fmt.Errorf("invalid maxlen value: %v", maxLen)
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("maxlen applies to strings only")
	}

	if len(str) > maxInt {
		return fmt.Errorf("length must be at most %d", maxInt)
	}

	return nil
}

// validateURL validates URL format
func validateURL(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("url must be a string")
	}

	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("invalid URL format")
	}

	return nil
}

// validatePhone validates phone number format
func validatePhone(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("phone must be a string")
	}

	// International phone number format
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(strings.ReplaceAll(str, " ", "")) {
		return fmt.Errorf("invalid phone number format")
	}

	return nil
}

// validateAlphanum validates alphanumeric characters
func validateAlphanum(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alphanum applies to strings only")
	}

	alphanumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumRegex.MatchString(str) {
		return fmt.Errorf("must contain only alphanumeric characters")
	}

	return nil
}

// validateNumeric validates numeric value
func validateNumeric(value interface{}, params map[string]interface{}) error {
	_, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("must be numeric")
	}
	return nil
}

// validateAlpha validates alphabetic characters
func validateAlpha(value interface{}, params map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("alpha applies to strings only")
	}

	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("must contain only alphabetic characters")
	}

	return nil
}

// Helper functions
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		// Trim whitespace
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, fmt.Errorf("cannot convert empty string to float64")
		}
		// Use strconv.ParseFloat which is stricter and validates the entire string
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to float64", v)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("cannot convert to float64")
	}
}

func toInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		// Trim whitespace
		v = strings.TrimSpace(v)
		if v == "" {
			return 0, fmt.Errorf("cannot convert empty string to int")
		}
		// Use strconv.Atoi which is stricter and validates the entire string
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("cannot convert '%s' to int", v)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("cannot convert to int")
	}
}
