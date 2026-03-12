package validator

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	if validator == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidator_ValidateRequired(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Name string `validate:"required"`
	}

	// Test empty value
	test := TestStruct{Name: ""}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid value
	test.Name = "test"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateEmail(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Email string `validate:"email"`
	}

	// Test invalid email
	test := TestStruct{Email: "invalid"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid email
	test.Email = "test@example.com"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateXSS(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Content string `validate:"xss"`
	}

	// Test XSS attempt
	test := TestStruct{Content: "<script>alert('xss')</script>"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid content
	test.Content = "Safe content"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateSQL(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Query string `validate:"sql"`
	}

	// Test SQL injection with UNION SELECT (most common pattern)
	test := TestStruct{Query: "UNION SELECT * FROM users"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors, "should detect UNION SELECT")

	// Test SQL injection with OR 1=1
	test.Query = "OR 1=1"
	errors = validator.Validate(test, "en")
	assert.NotEmpty(t, errors, "should detect OR 1=1")

	// Test safe content (no SQL keywords)
	test.Query = "normal user input"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors, "should allow safe content")
}

func TestValidator_ValidateMinMax(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Age int `validate:"min=18,max=100"`
	}

	// Test below minimum
	test := TestStruct{Age: 10}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test above maximum
	test.Age = 150
	errors = validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	test.Age = 25
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateMinLenMaxLen(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Password string `validate:"minlen=8,maxlen=20"`
	}

	// Test too short
	test := TestStruct{Password: "short"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test too long
	test.Password = "this is a very long password that exceeds the maximum length"
	errors = validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	test.Password = "validpass"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateURL(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		URL string `validate:"url"`
	}

	// Test invalid URL
	test := TestStruct{URL: "not a url"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid URL
	test.URL = "https://example.com"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidatePhone(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Phone string `validate:"phone"`
	}

	// Test invalid phone
	test := TestStruct{Phone: "invalid"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid phone
	test.Phone = "+1234567890"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateAlphanum(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Username string `validate:"alphanum"`
	}

	// Test invalid (contains special chars)
	test := TestStruct{Username: "user-name"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	test.Username = "username123"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateAlpha(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	type TestStruct struct {
		Name string `validate:"alpha"`
	}

	// Test invalid (contains numbers)
	test := TestStruct{Name: "name123"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	test.Name = "name"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_RegisterCustomRule(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	// Register custom rule
	validator.RegisterCustomRule("custom", func(value interface{}, params map[string]interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be string")
		}
		if str != "valid" {
			return fmt.Errorf("must be 'valid'")
		}
		return nil
	})

	type TestStruct struct {
		Field string `validate:"custom"`
	}

	// Test invalid
	test := TestStruct{Field: "invalid"}
	errors := validator.Validate(test, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	test.Field = "valid"
	errors = validator.Validate(test, "en")
	assert.Empty(t, errors)
}

func TestValidator_ValidateField(t *testing.T) {
	i18nManager := i18n.NewManager(nil)
	validator := NewValidator(i18nManager)

	rules := []Rule{
		{Name: "required", Func: validateRequired},
		{Name: "minlen", Func: validateMinLen, Params: map[string]interface{}{"value": "5"}},
	}

	// Test empty
	errors := validator.ValidateField("test", "", rules, "en")
	assert.NotEmpty(t, errors)

	// Test too short
	errors = validator.ValidateField("test", "abc", rules, "en")
	assert.NotEmpty(t, errors)

	// Test valid
	errors = validator.ValidateField("test", "valid", rules, "en")
	assert.Empty(t, errors)
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		&ValidationError{Field: "field1", Message: "error1"},
		&ValidationError{Field: "field2", Message: "error2"},
	}

	errMsg := errors.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
	if !strings.Contains(errMsg, "error1") {
		t.Error("expected error1 in message")
	}
}
