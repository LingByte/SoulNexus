package models

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBillingDB(t testing.TB) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&UsageRecord{}, &Bill{}, &Assistant{})
	require.NoError(t, err)

	return db
}

func TestCamelToSnake(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"camelCase", "camel_case"},
		{"PascalCase", "pascal_case"},
		{"simpleword", "simpleword"},
		{"HTTPSConnection", "https_connection"},
		{"XMLParser", "xml_parser"},
		{"userID", "user_id"},
		{"createdAt", "created_at"},
		{"", ""},
		{"A", "a"},
		{"AB", "ab"},
		{"ABC", "abc"},
		{"ABc", "a_bc"},
		{"AbC", "ab_c"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := camelToSnake(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertOrderBy(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"createdAt", "created_at"},
		{"createdAt DESC", "created_at DESC"},
		{"createdAt ASC", "created_at ASC"},
		{"userId DESC", "user_id DESC"},
		{"totalTokens ASC", "total_tokens ASC"},
		{"created_at", "created_at"}, // Already snake_case
		{"created_at DESC", "created_at DESC"},
		{"", ""},
		{"createdAt desc", "created_at DESC"}, // Case insensitive
		{"createdAt asc", "created_at ASC"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := convertOrderBy(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUsageRecord_TableName(t *testing.T) {
	record := UsageRecord{}
	assert.Equal(t, "usage_records", record.TableName())
}

func TestBill_TableName(t *testing.T) {
	bill := Bill{}
	assert.Equal(t, "bills", bill.TableName())
}

func TestUsageType_Constants(t *testing.T) {
	assert.Equal(t, UsageType("llm"), UsageTypeLLM)
	assert.Equal(t, UsageType("call"), UsageTypeCall)
	assert.Equal(t, UsageType("asr"), UsageTypeASR)
	assert.Equal(t, UsageType("tts"), UsageTypeTTS)
	assert.Equal(t, UsageType("api"), UsageTypeAPI)
}

func TestBillStatus_Constants(t *testing.T) {
	assert.Equal(t, BillStatus("draft"), BillStatusDraft)
	assert.Equal(t, BillStatus("generated"), BillStatusGenerated)
	assert.Equal(t, BillStatus("exported"), BillStatusExported)
	assert.Equal(t, BillStatus("archived"), BillStatusArchived)
}

func TestCreateUsageRecord(t *testing.T) {
	db := setupBillingDB(t)

	record := &UsageRecord{
		UserID:           1,
		CredentialID:     1,
		AssistantID:      uintPtr(1),
		SessionID:        "session-123",
		UsageType:        UsageTypeLLM,
		Model:            "gpt-3.5-turbo",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		UsageTime:        time.Now(),
	}

	err := CreateUsageRecord(db, record)
	assert.NoError(t, err)
	assert.NotZero(t, record.ID)
	assert.NotZero(t, record.CreatedAt)
	assert.NotZero(t, record.UpdatedAt)
}

func TestGetUsageRecords(t *testing.T) {
	db := setupBillingDB(t)

	// Create test records
	now := time.Now()
	records := []*UsageRecord{
		{
			UserID:       1,
			CredentialID: 1,
			AssistantID:  uintPtr(1),
			UsageType:    UsageTypeLLM,
			TotalTokens:  100,
			UsageTime:    now.Add(-2 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 2,
			AssistantID:  uintPtr(2),
			UsageType:    UsageTypeCall,
			CallDuration: 300,
			UsageTime:    now.Add(-1 * time.Hour),
		},
		{
			UserID:       2,
			CredentialID: 3,
			AssistantID:  uintPtr(1),
			UsageType:    UsageTypeLLM,
			TotalTokens:  200,
			UsageTime:    now.Add(-30 * time.Minute),
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	// Test basic query
	results, total, err := GetUsageRecords(db, 1, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Test filter by credential ID
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"credentialId": uint(1),
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, uint(1), results[0].CredentialID)

	// Test filter by usage type
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"usageType": UsageTypeLLM,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, UsageTypeLLM, results[0].UsageType)

	// Test pagination
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"page": 1,
		"size": 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 1)

	// Test time range filter
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"startTime": now.Add(-90 * time.Minute),
		"endTime":   now,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)

	// Test ordering
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"orderBy": "usageTime ASC",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)
	// First result should be older
	assert.True(t, results[0].UsageTime.Before(results[1].UsageTime))
}

func TestGetUsageStatistics(t *testing.T) {
	db := setupBillingDB(t)

	// Create test records
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	records := []*UsageRecord{
		{
			UserID:           1,
			CredentialID:     1,
			UsageType:        UsageTypeLLM,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			UsageTime:        now.Add(-2 * time.Hour),
		},
		{
			UserID:           1,
			CredentialID:     1,
			UsageType:        UsageTypeLLM,
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
			UsageTime:        now.Add(-1 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 300,
			CallCount:    1,
			UsageTime:    now.Add(-3 * time.Hour),
		},
		{
			UserID:        1,
			CredentialID:  1,
			UsageType:     UsageTypeASR,
			AudioDuration: 60,
			UsageTime:     now.Add(-4 * time.Hour),
		},
		{
			UserID:        1,
			CredentialID:  1,
			UsageType:     UsageTypeTTS,
			AudioDuration: 30,
			UsageTime:     now.Add(-5 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeAPI,
			APICallCount: 10,
			UsageTime:    now.Add(-6 * time.Hour),
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	// Get statistics
	credentialID := uint(1)
	stats, err := GetUsageStatistics(db, 1, startTime, endTime, &credentialID, nil)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify LLM stats
	assert.Equal(t, int64(2), stats.LLMCalls)
	assert.Equal(t, int64(450), stats.LLMTokens)        // 150 + 300
	assert.Equal(t, int64(300), stats.PromptTokens)     // 100 + 200
	assert.Equal(t, int64(150), stats.CompletionTokens) // 50 + 100

	// Verify call stats
	assert.Equal(t, int64(1), stats.CallCount)
	assert.Equal(t, int64(300), stats.CallDuration)
	assert.Equal(t, float64(300), stats.AvgCallDuration)

	// Verify ASR stats
	assert.Equal(t, int64(1), stats.ASRCount)
	assert.Equal(t, int64(60), stats.ASRDuration)

	// Verify TTS stats
	assert.Equal(t, int64(1), stats.TTSCount)
	assert.Equal(t, int64(30), stats.TTSDuration)

	// Verify API stats
	assert.Equal(t, int64(10), stats.APICalls)
}

func TestGetDailyUsageData(t *testing.T) {
	db := setupBillingDB(t)

	// Create test records for different days
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	records := []*UsageRecord{
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  100,
			UsageTime:    today.Add(2 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  200,
			UsageTime:    today.Add(4 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 300,
			UsageTime:    yesterday.Add(2 * time.Hour),
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	// Get daily usage data
	credentialID := uint(1)
	startTime := yesterday
	endTime := today.Add(23*time.Hour + 59*time.Minute)

	dailyData, err := GetDailyUsageData(db, 1, startTime, endTime, &credentialID, nil)
	assert.NoError(t, err)
	assert.Len(t, dailyData, 2) // Two days

	// Find today's data
	var todayData, yesterdayData *DailyUsageData
	for i := range dailyData {
		if strings.Contains(dailyData[i].Date, today.Format("2006-01-02")) {
			todayData = &dailyData[i]
		}
		if strings.Contains(dailyData[i].Date, yesterday.Format("2006-01-02")) {
			yesterdayData = &dailyData[i]
		}
	}

	require.NotNil(t, todayData)
	require.NotNil(t, yesterdayData)

	// Verify today's data
	assert.Equal(t, int64(2), todayData.LLMCalls)
	assert.Equal(t, int64(300), todayData.LLMTokens) // 100 + 200

	// Verify yesterday's data
	assert.Equal(t, int64(1), yesterdayData.CallCount)
	assert.Equal(t, int64(300), yesterdayData.CallDuration)
}

func TestCreateBill(t *testing.T) {
	db := setupBillingDB(t)

	bill := &Bill{
		UserID:                1,
		CredentialID:          uintPtr(1),
		BillNo:                "BILL-20230101120000-ABC123",
		Title:                 "Monthly Bill - January 2023",
		Status:                BillStatusGenerated,
		StartTime:             time.Now().Add(-30 * 24 * time.Hour),
		EndTime:               time.Now(),
		TotalLLMCalls:         100,
		TotalLLMTokens:        10000,
		TotalPromptTokens:     6000,
		TotalCompletionTokens: 4000,
		TotalCallDuration:     3600,
		TotalCallCount:        10,
		Notes:                 "Test bill",
	}

	err := CreateBill(db, bill)
	assert.NoError(t, err)
	assert.NotZero(t, bill.ID)
	assert.NotZero(t, bill.CreatedAt)
	assert.NotZero(t, bill.UpdatedAt)
}

func TestGetBills(t *testing.T) {
	db := setupBillingDB(t)

	// Create test bills
	now := time.Now()
	bills := []*Bill{
		{
			UserID:        1,
			CredentialID:  uintPtr(1),
			BillNo:        "BILL-001",
			Title:         "Bill 1",
			Status:        BillStatusGenerated,
			StartTime:     now.Add(-30 * 24 * time.Hour),
			EndTime:       now,
			TotalLLMCalls: 100,
		},
		{
			UserID:        1,
			CredentialID:  uintPtr(2),
			BillNo:        "BILL-002",
			Title:         "Bill 2",
			Status:        BillStatusExported,
			StartTime:     now.Add(-60 * 24 * time.Hour),
			EndTime:       now.Add(-30 * 24 * time.Hour),
			TotalLLMCalls: 200,
		},
		{
			UserID:        2,
			CredentialID:  uintPtr(3),
			BillNo:        "BILL-003",
			Title:         "Bill 3",
			Status:        BillStatusGenerated,
			StartTime:     now.Add(-30 * 24 * time.Hour),
			EndTime:       now,
			TotalLLMCalls: 50,
		},
	}

	for _, bill := range bills {
		err := CreateBill(db, bill)
		require.NoError(t, err)
	}

	// Test basic query
	results, total, err := GetBills(db, 1, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Test filter by credential ID
	results, total, err = GetBills(db, 1, map[string]interface{}{
		"credentialId": uint(1),
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "BILL-001", results[0].BillNo)

	// Test filter by status
	results, total, err = GetBills(db, 1, map[string]interface{}{
		"status": BillStatusExported,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, BillStatusExported, results[0].Status)

	// Test pagination
	results, total, err = GetBills(db, 1, map[string]interface{}{
		"page": 1,
		"size": 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 1)
}

func TestGetBill(t *testing.T) {
	db := setupBillingDB(t)

	// Create test bill
	bill := &Bill{
		UserID:        1,
		BillNo:        "BILL-SINGLE",
		Title:         "Single Bill Test",
		Status:        BillStatusGenerated,
		StartTime:     time.Now().Add(-30 * 24 * time.Hour),
		EndTime:       time.Now(),
		TotalLLMCalls: 150,
	}

	err := CreateBill(db, bill)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := GetBill(db, 1, bill.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "BILL-SINGLE", retrieved.BillNo)
	assert.Equal(t, "Single Bill Test", retrieved.Title)

	// Test not found (wrong user)
	retrieved, err = GetBill(db, 2, bill.ID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Test not found (wrong ID)
	retrieved, err = GetBill(db, 1, 999)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestUpdateBill(t *testing.T) {
	db := setupBillingDB(t)

	// Create initial bill
	bill := &Bill{
		UserID:        1,
		BillNo:        "BILL-UPDATE",
		Title:         "Update Test",
		Status:        BillStatusDraft,
		StartTime:     time.Now().Add(-30 * 24 * time.Hour),
		EndTime:       time.Now(),
		TotalLLMCalls: 100,
	}

	err := CreateBill(db, bill)
	require.NoError(t, err)

	// Update bill
	bill.Status = BillStatusGenerated
	bill.Title = "Updated Title"
	bill.TotalLLMCalls = 200
	bill.ExportFormat = "csv"
	bill.Notes = "Updated notes"

	err = UpdateBill(db, bill)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := GetBill(db, 1, bill.ID)
	require.NoError(t, err)
	assert.Equal(t, BillStatusGenerated, retrieved.Status)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, int64(200), retrieved.TotalLLMCalls)
	assert.Equal(t, "csv", retrieved.ExportFormat)
	assert.Equal(t, "Updated notes", retrieved.Notes)
}

func TestGenerateBillNo(t *testing.T) {
	billNo := GenerateBillNo()
	assert.NotEmpty(t, billNo)
	assert.True(t, strings.HasPrefix(billNo, "BILL-"))
	assert.Contains(t, billNo, time.Now().Format("20060102"))

	// Generate multiple bill numbers to ensure uniqueness
	billNos := make(map[string]bool)
	for i := 0; i < 100; i++ {
		billNo := GenerateBillNo()
		assert.False(t, billNos[billNo], "Bill number should be unique: %s", billNo)
		billNos[billNo] = true
	}
}

func TestRecordLLMUsage(t *testing.T) {
	db := setupBillingDB(t)

	err := RecordLLMUsage(db, 1, 1, uintPtr(1), uintPtr(1), "session-123", "gpt-3.5-turbo", 100, 50, 150)
	assert.NoError(t, err)

	// Verify record was created
	var record UsageRecord
	err = db.Where("user_id = ? AND usage_type = ?", 1, UsageTypeLLM).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, "gpt-3.5-turbo", record.Model)
	assert.Equal(t, 100, record.PromptTokens)
	assert.Equal(t, 50, record.CompletionTokens)
	assert.Equal(t, 150, record.TotalTokens)
}

func TestRecordCallUsage(t *testing.T) {
	db := setupBillingDB(t)

	callLogID := uint64(123)
	err := RecordCallUsage(db, 1, 1, uintPtr(1), uintPtr(1), "session-456", &callLogID, 300)
	assert.NoError(t, err)

	// Verify record was created
	var record UsageRecord
	err = db.Where("user_id = ? AND usage_type = ?", 1, UsageTypeCall).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, 300, record.CallDuration)
	assert.Equal(t, 1, record.CallCount)
	assert.NotNil(t, record.CallLogID)
	assert.Equal(t, uint64(123), *record.CallLogID)
}

func TestRecordASRUsage(t *testing.T) {
	db := setupBillingDB(t)

	err := RecordASRUsage(db, 1, 1, uintPtr(1), uintPtr(1), "session-789", 60, 1024)
	assert.NoError(t, err)

	// Verify record was created
	var record UsageRecord
	err = db.Where("user_id = ? AND usage_type = ?", 1, UsageTypeASR).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, 60, record.AudioDuration)
	assert.Equal(t, int64(1024), record.AudioSize)
}

func TestRecordTTSUsage(t *testing.T) {
	db := setupBillingDB(t)

	err := RecordTTSUsage(db, 1, 1, uintPtr(1), uintPtr(1), "session-abc", 30, 512)
	assert.NoError(t, err)

	// Verify record was created
	var record UsageRecord
	err = db.Where("user_id = ? AND usage_type = ?", 1, UsageTypeTTS).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, 30, record.AudioDuration)
	assert.Equal(t, int64(512), record.AudioSize)
}

func TestRecordAPIUsage(t *testing.T) {
	db := setupBillingDB(t)

	err := RecordAPIUsage(db, 1, 1, uintPtr(1), uintPtr(1), "session-def", 5, "API call description")
	assert.NoError(t, err)

	// Verify record was created
	var record UsageRecord
	err = db.Where("user_id = ? AND usage_type = ?", 1, UsageTypeAPI).First(&record).Error
	assert.NoError(t, err)
	assert.Equal(t, 5, record.APICallCount)
	assert.Equal(t, "API call description", record.Description)
}

func TestGenerateBill(t *testing.T) {
	db := setupBillingDB(t)

	// Create usage records first
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	records := []*UsageRecord{
		{
			UserID:           1,
			CredentialID:     1,
			UsageType:        UsageTypeLLM,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			UsageTime:        now.Add(-2 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 300,
			CallCount:    1,
			UsageTime:    now.Add(-1 * time.Hour),
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	// Generate bill
	credentialID := uint(1)
	bill, err := GenerateBill(db, 1, &credentialID, nil, startTime, endTime, "Test Generated Bill")
	assert.NoError(t, err)
	assert.NotNil(t, bill)
	assert.NotEmpty(t, bill.BillNo)
	assert.Equal(t, "Test Generated Bill", bill.Title)
	assert.Equal(t, BillStatusGenerated, bill.Status)
	assert.Equal(t, int64(1), bill.TotalLLMCalls)
	assert.Equal(t, int64(150), bill.TotalLLMTokens)
	assert.Equal(t, int64(100), bill.TotalPromptTokens)
	assert.Equal(t, int64(50), bill.TotalCompletionTokens)
	assert.Equal(t, int64(300), bill.TotalCallDuration)
	assert.Equal(t, int64(1), bill.TotalCallCount)
}

// Helper function to create uint pointer
func uintPtr(u uint) *uint {
	return &u
}

// Benchmark tests
func BenchmarkCreateUsageRecord(b *testing.B) {
	db := setupBillingDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &UsageRecord{
			UserID:       uint(i%10 + 1),
			CredentialID: uint(i%5 + 1),
			UsageType:    UsageTypeLLM,
			TotalTokens:  100,
			UsageTime:    time.Now(),
		}

		err := CreateUsageRecord(db, record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetUsageStatistics(b *testing.B) {
	db := setupBillingDB(b)

	// Create test data
	now := time.Now()
	for i := 0; i < 1000; i++ {
		record := &UsageRecord{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  100,
			UsageTime:    now.Add(-time.Duration(i) * time.Minute),
		}
		CreateUsageRecord(db, record)
	}

	credentialID := uint(1)
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetUsageStatistics(db, 1, startTime, endTime, &credentialID, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestUsageRecord_ComplexFiltering(t *testing.T) {
	db := setupBillingDB(t)

	// Create test records with different group IDs
	now := time.Now()
	groupID1 := uint(1)
	groupID2 := uint(2)

	records := []*UsageRecord{
		{
			UserID:       1,
			GroupID:      &groupID1,
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  100,
			UsageTime:    now,
		},
		{
			UserID:       1,
			GroupID:      &groupID2,
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  200,
			UsageTime:    now,
		},
		{
			UserID:       1,
			GroupID:      nil, // No group
			CredentialID: 1,
			UsageType:    UsageTypeLLM,
			TotalTokens:  300,
			UsageTime:    now,
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	// Test filter by group ID
	results, total, err := GetUsageRecords(db, 1, map[string]interface{}{
		"groupId": groupID1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, &groupID1, results[0].GroupID)

	// Test filter by group ID pointer
	results, total, err = GetUsageRecords(db, 1, map[string]interface{}{
		"groupId": &groupID2,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, &groupID2, results[0].GroupID)
}

func TestBill_ComplexFiltering(t *testing.T) {
	db := setupBillingDB(t)

	// Create test bills with different group IDs
	now := time.Now()
	groupID1 := uint(1)
	groupID2 := uint(2)

	bills := []*Bill{
		{
			UserID:    1,
			GroupID:   &groupID1,
			BillNo:    "BILL-GROUP1",
			Title:     "Group 1 Bill",
			Status:    BillStatusGenerated,
			StartTime: now.Add(-30 * 24 * time.Hour),
			EndTime:   now,
		},
		{
			UserID:    1,
			GroupID:   &groupID2,
			BillNo:    "BILL-GROUP2",
			Title:     "Group 2 Bill",
			Status:    BillStatusExported,
			StartTime: now.Add(-30 * 24 * time.Hour),
			EndTime:   now,
		},
		{
			UserID:    1,
			GroupID:   nil, // No group
			BillNo:    "BILL-NOGROUP",
			Title:     "No Group Bill",
			Status:    BillStatusGenerated,
			StartTime: now.Add(-30 * 24 * time.Hour),
			EndTime:   now,
		},
	}

	for _, bill := range bills {
		err := CreateBill(db, bill)
		require.NoError(t, err)
	}

	// Test filter by group ID
	results, total, err := GetBills(db, 1, map[string]interface{}{
		"groupId": groupID1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "BILL-GROUP1", results[0].BillNo)

	// Test filter by group ID pointer
	results, total, err = GetBills(db, 1, map[string]interface{}{
		"groupId": &groupID2,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "BILL-GROUP2", results[0].BillNo)
}

func TestUsageStatistics_EdgeCases(t *testing.T) {
	db := setupBillingDB(t)

	// Test with no data
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	stats, err := GetUsageStatistics(db, 1, startTime, endTime, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.LLMCalls)
	assert.Equal(t, int64(0), stats.CallCount)
	assert.Equal(t, float64(0), stats.AvgCallDuration)

	// Test with multiple calls for average calculation
	records := []*UsageRecord{
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 100,
			CallCount:    1,
			UsageTime:    now.Add(-1 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 200,
			CallCount:    1,
			UsageTime:    now.Add(-2 * time.Hour),
		},
		{
			UserID:       1,
			CredentialID: 1,
			UsageType:    UsageTypeCall,
			CallDuration: 300,
			CallCount:    1,
			UsageTime:    now.Add(-3 * time.Hour),
		},
	}

	for _, record := range records {
		err := CreateUsageRecord(db, record)
		require.NoError(t, err)
	}

	stats, err = GetUsageStatistics(db, 1, startTime, endTime, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), stats.CallCount)
	assert.Equal(t, int64(600), stats.CallDuration)      // 100 + 200 + 300
	assert.Equal(t, float64(200), stats.AvgCallDuration) // 600 / 3
}
