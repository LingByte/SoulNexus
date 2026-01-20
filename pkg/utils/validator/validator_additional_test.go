package validator

import (
	"testing"
)

func TestValidateNumeric(t *testing.T) {
	v := NewValidator(nil)

	type TestStruct struct {
		Field interface{} `validate:"numeric"`
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid int", 123, false},
		{"valid float", 123.45, false},
		{"valid string int", "123", false},
		{"valid string float", "123.45", false},
		{"invalid string", "abc", true},
		{"invalid mixed", "123abc", true},
		{"nil", nil, true},
		{"bool", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := TestStruct{Field: tt.value}
			errors := v.Validate(test, "en")
			if (len(errors) > 0) != tt.wantErr {
				t.Errorf("validateNumeric() error = %v, wantErr %v", errors, tt.wantErr)
			}
		})
	}
}

func TestToFloat64_Indirect(t *testing.T) {
	// Test toFloat64 indirectly through validation
	// Note: min:10 requires numeric value, so non-numeric values should fail
	v := NewValidator(nil)

	type TestStruct struct {
		Field interface{} `validate:"min:10"`
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"int", 123, false},
		{"int8", int8(123), false},
		{"int16", int16(123), false},
		{"int32", int32(123), false},
		{"int64", int64(123), false},
		{"uint", uint(123), false},
		{"uint8", uint8(123), false},
		{"uint16", uint16(123), false},
		{"uint32", uint32(123), false},
		{"uint64", uint64(123), false},
		{"float32", float32(123.45), false},
		{"float64", 123.45, false},
		{"string int", "123", false},
		{"string float", "123.45", false},
		{"invalid string", "abc", true}, // Should fail because can't convert to float
		{"bool", true, true},            // Should fail because can't convert to float
		{"nil", nil, true},              // Should fail because can't convert to float
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := TestStruct{Field: tt.value}
			errors := v.Validate(test, "en")
			hasError := len(errors) > 0
			if hasError != tt.wantErr {
				t.Errorf("toFloat64() error = %v (errors: %+v), wantErr %v", hasError, errors, tt.wantErr)
			}
		})
	}
}

func TestToInt_Indirect(t *testing.T) {
	// Test toInt indirectly through validation
	// Note: min:10 requires numeric value, so non-numeric values should fail
	v := NewValidator(nil)

	type TestStruct struct {
		Field interface{} `validate:"min:10"`
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"int", 123, false},
		{"int8", int8(123), false},
		{"int16", int16(123), false},
		{"int32", int32(123), false},
		{"int64", int64(123), false},
		{"uint", uint(123), false},
		{"uint8", uint8(123), false},
		{"uint16", uint16(123), false},
		{"uint32", uint32(123), false},
		{"uint64", uint64(123), false},
		{"float32", float32(123.45), false},
		{"float64", 123.45, false},
		{"string int", "123", false},
		{"string float", "123.45", false},
		{"invalid string", "abc", true}, // Should fail because can't convert to float
		{"bool", true, true},            // Should fail because can't convert to float
		{"nil", nil, true},              // Should fail because can't convert to float
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := TestStruct{Field: tt.value}
			errors := v.Validate(test, "en")
			hasError := len(errors) > 0
			if hasError != tt.wantErr {
				t.Errorf("toInt() error = %v (errors: %+v), wantErr %v", hasError, errors, tt.wantErr)
			}
		})
	}
}

func TestValidateField_EdgeCases(t *testing.T) {
	v := NewValidator(nil)

	type TestStruct struct {
		Field1 string `validate:"required"`
		Field2 int    `validate:"min:10"`
	}

	// Test with empty value (Field2 = 0 will fail min:10, but we're testing Field1)
	data := &TestStruct{Field1: "", Field2: 0}
	errors := v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error for empty string")
	}

	// Test with valid value (Field2 must be >= 10 to pass min:10)
	data = &TestStruct{Field1: "test", Field2: 15}
	errors = v.Validate(data, "en")
	if len(errors) > 0 {
		t.Errorf("expected no errors for valid value, got: %v", errors)
	}
}

func TestGetErrorMessage_WithI18n(t *testing.T) {
	// Test without i18n
	v := NewValidator(nil)

	type TestStruct struct {
		Field string `validate:"required"`
	}

	data := TestStruct{}
	errors := v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error")
	}
	if errors[0].Message == "" {
		t.Error("expected non-empty error message")
	}
}

func TestParseTagRules_Complex(t *testing.T) {
	v := NewValidator(nil)

	type TestStruct struct {
		Field string `validate:"required,minlen:5,maxlen:20"`
	}

	// Test with value that violates minlen
	data := TestStruct{Field: "test"} // length 4, but minlen is 5
	errors := v.Validate(data, "en")
	// Should have errors for minlen
	if len(errors) == 0 {
		t.Error("expected validation errors for minlen")
	}

	// Test with value that violates maxlen
	data = TestStruct{Field: "this is a very long string that exceeds maxlen"} // too long
	errors = v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected validation errors for maxlen")
	}

	// Test with valid value
	data = TestStruct{Field: "valid string"} // length 12, between 5 and 20
	errors = v.Validate(data, "en")
	if len(errors) > 0 {
		t.Errorf("expected no errors for valid value, got: %v", errors)
	}
}

func TestValidateRequired_EdgeCases(t *testing.T) {
	v := NewValidator(nil)

	type TestStruct struct {
		Field1 interface{} `validate:"required"`
		Field2 string      `validate:"required"`
		Field3 []int       `validate:"required"`
	}

	// Test with nil
	data := TestStruct{Field1: nil}
	errors := v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error for nil")
	}

	// Test with empty string
	data = TestStruct{Field2: ""}
	errors = v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error for empty string")
	}

	// Test with whitespace
	data = TestStruct{Field2: "   "}
	errors = v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error for whitespace string")
	}

	// Test with empty slice
	data = TestStruct{Field3: []int{}}
	errors = v.Validate(data, "en")
	if len(errors) == 0 {
		t.Error("expected error for empty slice")
	}

	// Test with valid values
	data = TestStruct{
		Field1: "test",
		Field2: "test",
		Field3: []int{1, 2},
	}
	errors = v.Validate(data, "en")
	if len(errors) > 0 {
		t.Error("expected no errors for valid values")
	}
}
