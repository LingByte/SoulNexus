package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInjectDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	router := gin.New()
	router.Use(InjectDB(db))
	router.GET("/test", func(c *gin.Context) {
		dbFromCtx := c.MustGet(constants.DbField).(*gorm.DB)
		assert.NotNil(t, dbFromCtx)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInjectDB_NilDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(InjectDB(nil))
	router.GET("/test", func(c *gin.Context) {
		dbFromCtx, exists := c.Get(constants.DbField)
		if exists {
			assert.Nil(t, dbFromCtx)
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
