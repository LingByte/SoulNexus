package petproject_test

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeFileContent_Text(t *testing.T) {
	b, err := petproject.DecodeFileContent("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
}

func TestDecodeFileContent_Base64(t *testing.T) {
	encoded := petproject.BinaryPrefix + "aGVsbG8="
	b, err := petproject.DecodeFileContent(encoded)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
}

func TestEncodeFileForAPI_BinaryPath(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47}
	got := petproject.EncodeFileForAPI("assets/model/t.png", raw)
	assert.True(t, len(got) > len(petproject.BinaryPrefix))
	assert.Equal(t, raw, mustDecode(t, got))
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := petproject.DecodeFileContent(s)
	require.NoError(t, err)
	return b
}
