// Package encryption provides the encryption and decryption functions for the key-value store.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

type Encryptor struct {
	key []byte
}

func New(key []byte) (*Encryptor, error) {
	if key == nil {
		key = []byte(os.Getenv("SKVS_ENCRYPTION_KEY"))
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("Key must be exactly 32 bytes for AES-256-GCM")
	}

	return &Encryptor{key: key}, nil
}

func (e *Encryptor) Encrypt(payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("encryption: new cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption: new gcm: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encryption: nonce: %v", err)
	}

	return gcm.Seal(nonce, nonce, payload, nil), nil
}

func (e *Encryptor) Decrypt(payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("decryption: new cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("decryption: new gcm: %v", err)
	}

	if len(payload) < gcm.NonceSize() {
		return nil, fmt.Errorf("decryption: ciphertext too short")
	}

	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption: %v", err)
	}

	return plaintext, nil
}
