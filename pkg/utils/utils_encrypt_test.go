package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD5(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		expected string
	}{
		{
			name:     "Test MD5 with string",
			str:      "hello",
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name:     "Test MD5 with empty string",
			str:      "",
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "Test MD5 with special characters",
			str:      "!@#$%^&*()",
			expected: "05b28d17a7b6e7024b6e5d8cc43a8bf7",
		},
		{
			name:     "Test MD5 with unicode characters",
			str:      "你好世界",
			expected: "65396ee4aad0b4f17aacd1c6112ee364",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := MD5(tt.str)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestRC4EncryptAndDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		key       []byte
		plaintext string
	}{
		{
			name:      "Test RC4 Encrypt and Decrypt with valid input",
			key:       []byte("1234567890123456"),
			plaintext: "1234567890123456",
		},
		{
			name:      "Test RC4 Encrypt and Decrypt with empty input",
			key:       []byte("1234567890123456"),
			plaintext: "",
		},
		{
			name:      "Test RC4 Encrypt and Decrypt with random input",
			key:       []byte("randomkey123456"),
			plaintext: "testplaintext",
		},
		{
			name:      "Test RC4 Encrypt and Decrypt with unicode",
			key:       []byte("unicodekey123456"),
			plaintext: "你好世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Encryption
			ciphertext := RC4Encrypt(tt.plaintext, tt.key)

			// Test Decryption
			decryptedText := RC4Decrypt(ciphertext, tt.key)

			// Ensure the decrypted text matches the original plaintext
			assert.Equal(t, tt.plaintext, decryptedText)
		})
	}
}

func TestAesEncryptEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		key       []byte
		plaintext []byte
		expectErr bool
	}{
		{
			name:      "Test AES Encrypt with short key",
			key:       []byte("shortkey"),
			plaintext: []byte("testplaintext"),
			expectErr: true,
		},
		{
			name:      "Test AES Encrypt with long key (24 bytes)",
			key:       []byte("123456789012345678901234"), // 24-byte key
			plaintext: []byte("testplaintext"),
			expectErr: false,
		},
		{
			name:      "Test AES Encrypt with long plaintext",
			key:       []byte("1234567890123456"),
			plaintext: make([]byte, 1024*1024*10), // 10MB data
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AesEncrypt(tt.key, tt.plaintext)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAesDecryptEdgeCases(t *testing.T) {
	// Test with valid ciphertext
	key := []byte("1234567890123456")
	plaintext := []byte("testplaintext")

	ciphertext, err := AesEncrypt(key, plaintext)
	assert.NoError(t, err)

	// Test normal decryption
	_, err = AesDecrypt(key, ciphertext)
	assert.NoError(t, err)

	// Test with short ciphertext
	shortCiphertext := []byte("short")
	_, err = AesDecrypt(key, shortCiphertext)
	assert.Error(t, err)

	// Test with wrong key (this should not error but produce garbage)
	wrongKey := []byte("wrongkey12345678")
	_, err = AesDecrypt(wrongKey, ciphertext)
	// We don't assert on error here because AES decryption with wrong key
	// typically doesn't produce an error, just garbage output
	assert.NoError(t, err) // Decryption may not error but will produce garbage
}

func TestRC4EncryptEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		key       []byte
		plaintext string
	}{
		{
			name:      "Test RC4 Encrypt with short key",
			key:       []byte("shortkey"),
			plaintext: "testplaintext",
		},
		{
			name:      "Test RC4 Encrypt with long key",
			key:       make([]byte, 256),
			plaintext: "testplaintext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// RC4 should handle various key sizes
			result := RC4Encrypt(tt.plaintext, tt.key)
			// Empty key now returns empty result, so we only check non-empty keys
			if len(tt.key) > 0 {
				assert.NotEmpty(t, result)

				// Verify we can decrypt it back
				decrypted := RC4Decrypt(result, tt.key)
				assert.Equal(t, tt.plaintext, decrypted)
			} else {
				// With empty key, we expect empty result
				assert.Equal(t, "", result)
			}
		})
	}

	// Test special case: empty key
	t.Run("Test RC4 Encrypt with empty key", func(t *testing.T) {
		key := []byte("")
		plaintext := "testplaintext"

		// RC4 with empty key should return empty string
		result := RC4Encrypt(plaintext, key)
		assert.Equal(t, "", result)

		// Decrypting empty string should also return empty string
		decrypted := RC4Decrypt(result, key)
		assert.Equal(t, "", decrypted)
	})
}

func TestRC4DecryptEdgeCases(t *testing.T) {
	// Test with valid encrypted data
	key := []byte("testkey123456789")
	plaintext := "testplaintext"

	encrypted := RC4Encrypt(plaintext, key)
	decrypted := RC4Decrypt(encrypted, key)
	assert.Equal(t, plaintext, decrypted)

	// Test with invalid hex string
	invalidResult := RC4Decrypt("invalidhex", key)
	assert.Equal(t, "", invalidResult)

	// Test with empty string
	emptyResult := RC4Decrypt("", key)
	assert.Equal(t, "", emptyResult)

	// Test with wrong key
	wrongKey := []byte("wrongkey12345678")
	wrongResult := RC4Decrypt(encrypted, wrongKey)
	assert.NotEqual(t, plaintext, wrongResult)
}
