package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactory_DefaultProviderRequiresOptions(t *testing.T) {
	_, err := New("", nil)
	assert.Error(t, err)
}

func TestFactory_Qdrant(t *testing.T) {
	h, err := New("qdrant", &FactoryOptions{Qdrant: &QdrantOptions{BaseURL: "http://localhost:6333", Collection: "c", Embedder: &NvidiaEmbedClient{}}})
	assert.NoError(t, err)
	assert.NotNil(t, h)
	assert.Equal(t, KnowledgeQdrant, h.Provider())
}

func TestFactory_Unsupported(t *testing.T) {
	_, err := New("nope", &FactoryOptions{})
	assert.ErrorIs(t, err, ErrUnsupportedKnowledgeProvider)
}
