package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCipher(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			key:     "12345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "invalid key too short",
			key:     "short-key",
			wantErr: true,
		},
		{
			name:    "invalid key too long",
			key:     "1234567890123456789012345678901234567890",
			wantErr: true,
		},
		{
			name:    "invalid key 16 bytes",
			key:     "1234567890123456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cipher, err := NewCipher(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cipher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cipher)
			}
		})
	}
}

func TestCipherEncryptDecrypt(t *testing.T) {
	key := "12345678901234567890123456789012"
	cipher, err := NewCipher(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "json config",
			plaintext: `{"host":"localhost","port":3306,"user":"root","password":"secret123"}`,
		},
		{
			name:      "unicode text",
			plaintext: "你好，世界！🌍 émojis",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long text",
			plaintext: `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`,
		},
		{
			name:      "special chars",
			plaintext: `!@#$%^&*()_+-=[]{}|;':",./<>?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := cipher.Encrypt(tt.plaintext)
			require.NoError(t, err)

			// Empty plaintext should return empty ciphertext
			if tt.plaintext == "" {
				assert.Empty(t, encrypted)
				return
			}

			// Encrypted text should be different from plaintext
			assert.NotEqual(t, tt.plaintext, encrypted)

			// Decrypt
			decrypted, err := cipher.Decrypt(encrypted)
			require.NoError(t, err)

			// Decrypted text should match original
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestCipherDecryptWithWrongKey(t *testing.T) {
	key1 := "12345678901234567890123456789012"
	key2 := "09876543210987654321098765432109"

	cipher1, err := NewCipher(key1)
	require.NoError(t, err)

	cipher2, err := NewCipher(key2)
	require.NoError(t, err)

	plaintext := "sensitive data"
	encrypted, err := cipher1.Encrypt(plaintext)
	require.NoError(t, err)

	// Try to decrypt with different key
	_, err = cipher2.Decrypt(encrypted)
	assert.Error(t, err)
}

func TestCipherDecryptInvalidData(t *testing.T) {
	key := "12345678901234567890123456789012"
	cipher, err := NewCipher(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "invalid base64",
			ciphertext: "!!!not-valid-base64!!!",
		},
		{
			name:       "too short data",
			ciphertext: "dG9vc2hvcnQ=", // base64 of "tooshort"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cipher.Decrypt(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestCipherNonceUniqueness(t *testing.T) {
	key := "12345678901234567890123456789012"
	cipher, err := NewCipher(key)
	require.NoError(t, err)

	plaintext := "same plaintext"

	// Encrypt same plaintext multiple times
	encrypted1, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted3, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)

	// Each encryption should produce different ciphertext (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2)
	assert.NotEqual(t, encrypted2, encrypted3)
	assert.NotEqual(t, encrypted1, encrypted3)

	// But all should decrypt to same plaintext
	decrypted1, err := cipher.Decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := cipher.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)

	decrypted3, err := cipher.Decrypt(encrypted3)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted3)
}
