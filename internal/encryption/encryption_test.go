package encryption

import (
	"bytes"
	"testing"
)

func TestNew(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012") // 32 bytes

	t.Run("valid key", func(t *testing.T) {
		enc, err := New(validKey)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if enc == nil {
			t.Fatal("expected encryptor, got nil")
		}
	})

	t.Run("invalid key length", func(t *testing.T) {
		shortKey := []byte("tooshort")
		_, err := New(shortKey)
		if err == nil {
			t.Fatal("expected error for short key, got nil")
		}
	})

	t.Run("nil key uses environment", func(t *testing.T) {
		t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

		enc, err := New(nil)
		if err != nil {
			t.Fatalf("expected no error with env key, got %v", err)
		}
		if enc == nil {
			t.Fatal("expected encryptor, got nil")
		}
	})

	t.Run("nil key with missing env", func(t *testing.T) {
		t.Setenv("SKVS_ENCRYPTION_KEY", "") // ensure it's empty

		_, err := New(nil)
		if err == nil {
			t.Fatal("expected error when key is nil and env is empty")
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	enc, err := New(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	testCases := []struct {
		name    string
		payload []byte
	}{
		{"simple string", []byte("hello world")},
		{"empty payload", []byte("")},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tc.payload)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			decrypted, err := enc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if !bytes.Equal(decrypted, tc.payload) {
				t.Errorf("decrypted data mismatch: got %v, want %v", decrypted, tc.payload)
			}
		})
	}
}

func TestDecryptInvalidData(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	enc, err := New(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	t.Run("too short payload", func(t *testing.T) {
		_, err := enc.Decrypt([]byte("short"))
		if err == nil {
			t.Fatal("expected error for short payload, got nil")
		}
	})

	t.Run("corrupted data", func(t *testing.T) {
		encrypted, _ := enc.Encrypt([]byte("test data"))
		corrupted := append([]byte{}, encrypted...)
		corrupted[len(corrupted)-1] ^= 0xFF // flip bits

		_, err := enc.Decrypt(corrupted)
		if err == nil {
			t.Fatal("expected error for corrupted data, got nil")
		}
	})
}
