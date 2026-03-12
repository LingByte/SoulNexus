package validator

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// Middleware creates a Gin middleware for validation
func Middleware(validator *Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("validator", validator)
		c.Next()
	}
}

// ValidateStruct validates a struct and returns errors
func ValidateStruct(c *gin.Context, data interface{}) ValidationErrors {
	validator, _ := c.Get("validator")
	if v, ok := validator.(*Validator); ok {
		locale := i18n.GetLocaleFromGin(c)
		return v.Validate(data, locale)
	}
	return ValidationErrors{}
}

// ShouldBindJSON validates and binds JSON
func ShouldBindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return err
	}

	errors := ValidateStruct(c, obj)
	if len(errors) > 0 {
		response.Fail(c, "validation failed", errors)
		return errors
	}

	return nil
}

// ShouldBindQuery validates and binds query parameters
func ShouldBindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return err
	}

	errors := ValidateStruct(c, obj)
	if len(errors) > 0 {
		response.Fail(c, "validation failed", errors)
		return errors
	}

	return nil
}

// ShouldBindForm validates and binds form data
func ShouldBindForm(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBind(obj); err != nil {
		return err
	}

	errors := ValidateStruct(c, obj)
	if len(errors) > 0 {
		response.Fail(c, "validation failed", errors)
		return errors
	}

	return nil
}
