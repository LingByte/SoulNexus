package rerank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLKEAPRerankClient_requiresCredentials(t *testing.T) {
	_, err := NewLKEAPRerankClient(&RerankClientConfig{
		Model: "lke-reranker-base",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret_id")
}

func TestNewLKEAPRerankClient_secretKeyFromExtraConfig(t *testing.T) {
	r, err := NewLKEAPRerankClient(&RerankClientConfig{
		APIKey: "AKIDtest",
		Model:  "lke-reranker-base",
		ExtraConfig: map[string]string{
			"secret_key": "sk-test",
			"region":     "ap-beijing",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "lke-reranker-base", r.Model)
}

func TestNewLKEAPRerankClient_secretKeyField(t *testing.T) {
	r, err := NewLKEAPRerankClient(&RerankClientConfig{
		APIKey:    "AKIDtest",
		SecretKey: "sk-test",
	})
	require.NoError(t, err)
	assert.Equal(t, lkeapDefaultRerankModel, r.Model)
}

func TestLKEAPRerankClient_Rerank_tooManyDocuments(t *testing.T) {
	r, err := NewLKEAPRerankClient(&RerankClientConfig{
		APIKey:    "AKIDtest",
		SecretKey: "sk-test",
	})
	require.NoError(t, err)

	docs := make([]string, 61)
	for i := range docs {
		docs[i] = "doc"
	}
	_, err = r.Rerank(t.Context(), "query", docs, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "60")
}
