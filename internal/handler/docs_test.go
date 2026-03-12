package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/internal/apidocs"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandlers_GetDocs(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create handlers instance
	handlers := &Handlers{db: db}

	// Test GetDocs method directly
	docs := handlers.GetDocs()

	// Verify that docs are returned
	assert.NotNil(t, docs)
	assert.IsType(t, []apidocs.UriDoc{}, docs)
	assert.Greater(t, len(docs), 0, "Should return at least one doc")
}

func TestHandlers_DocsHTTPHandler(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create handlers instance
	handlers := &Handlers{db: db}

	// Setup Gin router with apidocs registration
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register docs handlers using apidocs package
	uriDocs := handlers.GetDocs()
	objDocs := []apidocs.WebObjectDoc{} // Empty for test
	apidocs.RegisterHandler("/docs", router, uriDocs, objDocs, db)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "Get API documentation HTML",
			path:           "/docs",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
		},
		{
			name:           "Get API documentation JSON",
			path:           "/docs.json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), tt.expectedType)
			assert.NotEmpty(t, w.Body.String())
		})
	}
}

func TestHandlers_GetObjs(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create handlers instance
	handlers := &Handlers{db: db}

	// Test GetObjs method
	objs := handlers.GetObjs()

	// Verify that objects are returned
	assert.NotNil(t, objs)
	assert.Greater(t, len(objs), 0, "Should return at least one object")
}

func BenchmarkHandlers_GetDocs(b *testing.B) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(b, err)

	// Create handlers instance
	handlers := &Handlers{db: db}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs := handlers.GetDocs()
		_ = docs
	}
}

func BenchmarkHandlers_GetObjs(b *testing.B) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(b, err)

	// Create handlers instance
	handlers := &Handlers{db: db}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		objs := handlers.GetObjs()
		_ = objs
	}
}
