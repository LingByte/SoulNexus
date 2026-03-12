package knowledge

import (
	"context"
	"mime/multipart"
	"os"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

const (
	envBailianAccessKeyID     = "BAILIAN_ACCESS_KEY_ID"
	envBailianAccessKeySecret = "BAILIAN_ACCESS_KEY_SECRET"
	envBailianEndpoint        = "BAILIAN_ENDPOINT"
	envBailianWorkspaceID     = "BAILIAN_WORKSPACE_ID"
	envBailianCategoryID      = "BAILIAN_CATEGORY_ID"
)

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		ConfigKeyAliyunAccessKeyID:     utils.GetEnv(envBailianAccessKeyID),
		ConfigKeyAliyunAccessKeySecret: utils.GetEnv(envBailianAccessKeySecret),
		ConfigKeyAliyunEndpoint:        utils.GetEnv(envBailianEndpoint),
		ConfigKeyAliyunWorkspaceID:     utils.GetEnv(envBailianWorkspaceID),
		ConfigKeyAliyunCategoryID:      utils.GetEnv(envBailianCategoryID),
		ConfigKeyAliyunSourceType:      utils.GetEnv("BAILIAN_SOURCE_TYPE"),
		ConfigKeyAliyunParser:          utils.GetEnv("BAILIAN_PARSER"),
		ConfigKeyAliyunStructType:      utils.GetEnv("BAILIAN_STRUCT_TYPE"),
		ConfigKeyAliyunSinkType:        utils.GetEnv("BAILIAN_SINK_TYPE"),
	}
}

func hasTestCredentials() bool {
	return utils.GetEnv(envBailianAccessKeyID) != "" &&
		utils.GetEnv(envBailianAccessKeySecret) != "" &&
		utils.GetEnv(envBailianWorkspaceID) != "" &&
		utils.GetEnv(envBailianCategoryID) != ""
}

func TestNewAliyunKnowledgeBase(t *testing.T) {
	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create Aliyun knowledge base: %v", err)
	}
	if kb.Provider() != ProviderAliyun {
		t.Errorf("expected provider %s, got %s", ProviderAliyun, kb.Provider())
	}
}

func TestNewAliyunKnowledgeBase_MissingCredentials(t *testing.T) {
	config := map[string]interface{}{
		ConfigKeyAliyunAccessKeyID:     "",
		ConfigKeyAliyunAccessKeySecret: "",
	}
	_, err := NewAliyunKnowledgeBase(config)
	if err == nil {
		t.Error("expected error for missing credentials, got nil")
	}
}

func TestAliyunProvider(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}
	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}
	if kb.Provider() != ProviderAliyun {
		t.Errorf("expected provider %s, got %s", ProviderAliyun, kb.Provider())
	}
}

func TestAliyunCreateIndex(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()
	indexName := "test-index-" + utils.GetEnv("USER")

	indexID, err := kb.CreateIndex(ctx, indexName, nil)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	if indexID == "" {
		t.Error("expected non-empty index ID")
	}

	t.Logf("Created index with ID: %s", indexID)
	kb.DeleteIndex(ctx, indexID)
}

func TestAliyunCreateIndex_WithConfig(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()
	indexName := "test-index-with-config"

	configParams := map[string]interface{}{
		ConfigKeyAliyunStructType: utils.GetEnv("BAILIAN_STRUCT_TYPE"),
		ConfigKeyAliyunSourceType: utils.GetEnv("BAILIAN_SOURCE_TYPE"),
		ConfigKeyAliyunSinkType:   utils.GetEnv("BAILIAN_SINK_TYPE"),
	}

	indexID, err := kb.CreateIndex(ctx, indexName, configParams)
	if err != nil {
		t.Fatalf("failed to create index with config: %v", err)
	}

	if indexID == "" {
		t.Error("expected non-empty index ID")
	}

	t.Logf("Created index with config, ID: %s", indexID)

	defer kb.DeleteIndex(ctx, indexID)
}

func TestAliyunDeleteIndex(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()

	// Create an index first
	indexName := "test-index-to-delete"
	indexID, err := kb.CreateIndex(ctx, indexName, nil)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Then delete it
	err = kb.DeleteIndex(ctx, indexID)
	if err != nil {
		t.Fatalf("failed to delete index: %v", err)
	}

	t.Log("Successfully deleted index")
}

func TestAliyunSearch(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()

	// Create an index first
	indexName := "test-index-search"
	indexID, err := kb.CreateIndex(ctx, indexName, nil)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	defer kb.DeleteIndex(ctx, indexID)

	// Search with the created index
	options := SearchOptions{
		Query: "test query",
		TopK:  5,
	}

	results, err := kb.Search(ctx, indexID, options)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	// Results may be empty if index has no documents, which is expected
	t.Logf("Search returned %d results", len(results))
}

func TestAliyunUploadDocument_WithRealFile(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	content := []byte("This is a test document.\nIt has multiple lines.\nFor testing upload functionality.")
	tmpFile, err := os.CreateTemp("", "test-doc-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()
	knowledgeKey := "test-real-file-" + utils.GetEnv("USER")

	header := &multipart.FileHeader{
		Filename: tmpFile.Name(),
		Size:     int64(len(content)),
	}

	err = kb.UploadDocument(ctx, knowledgeKey, file, header, map[string]interface{}{
		"source": "real_file_test",
	})
	if err != nil {
		t.Fatalf("failed to upload document: %v", err)
	}

	t.Log("Successfully uploaded real file")

	defer kb.DeleteIndex(ctx, knowledgeKey)
}

func TestAliyunDeleteDocument(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()

	// Aliyun does not support deleting individual documents
	err = kb.DeleteDocument(ctx, "any-knowledge-key", "any-document-id")
	if err == nil {
		t.Error("expected error when deleting document, got nil")
	}
}

func TestAliyunListDocuments(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()

	// Aliyun does not support listing documents
	_, err = kb.ListDocuments(ctx, "any-knowledge-key")
	if err == nil {
		t.Error("expected error when listing documents, got nil")
	}
}

func TestAliyunGetDocument(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()

	// Aliyun does not support getting document content
	_, err = kb.GetDocument(ctx, "any-knowledge-key", "any-document-id")
	if err == nil {
		t.Error("expected error when getting document, got nil")
	}
}

func TestAliyunIntegration_FullWorkflow(t *testing.T) {
	if !hasTestCredentials() {
		t.Skip("missing test credentials")
	}

	config := getTestConfig()
	kb, err := NewAliyunKnowledgeBase(config)
	if err != nil {
		t.Fatalf("failed to create knowledge base: %v", err)
	}

	ctx := context.Background()
	knowledgeKey := "test-integration-" + utils.GetEnv("USER")

	// Create a test file
	content := []byte("Integration test document content.\nThis is used for full workflow testing.")
	tmpFile, err := os.CreateTemp("", "integration-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.Write(content)
	tmpFile.Close()

	file, _ := os.Open(tmpFile.Name())
	header := &multipart.FileHeader{
		Filename: tmpFile.Name(),
		Size:     int64(len(content)),
	}

	// Upload document (creates index)
	t.Log("Uploading document...")
	err = kb.UploadDocument(ctx, knowledgeKey, file, header, map[string]interface{}{
		"test": "integration",
	})
	if err != nil {
		t.Fatalf("upload document failed: %v", err)
	}

	// Search
	t.Log("Searching...")
	options := SearchOptions{
		Query: "integration test",
		TopK:  5,
	}

	results, err := kb.Search(ctx, knowledgeKey, options)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	t.Logf("Search returned %d results", len(results))

	// Delete index
	t.Log("Deleting index...")
	err = kb.DeleteIndex(ctx, knowledgeKey)
	if err != nil {
		t.Fatalf("failed to delete index: %v", err)
	}

	t.Log("Integration test completed successfully")
}
