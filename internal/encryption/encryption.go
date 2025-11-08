// Package encryption provides the encryption and decryption functions for the key-value store.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type Encryptor struct {
	key   []byte
	block cipher.Block
	gcm   cipher.AEAD
}

func New(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be exactly 32 bytes for AES-256-GCM")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encryption: new cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption: new gcm: %v", err)
	}

	return &Encryptor{key: key, block: block, gcm: gcm}, nil
}

func (e *Encryptor) Encrypt(payload []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encryption: nonce: %v", err)
	}

	return e.gcm.Seal(nonce, nonce, payload, nil), nil
}

func (e *Encryptor) Decrypt(payload []byte) ([]byte, error) {
	if len(payload) < e.gcm.NonceSize() {
		return nil, fmt.Errorf("decryption: ciphertext too short")
	}

	nonce, ciphertext := payload[:e.gcm.NonceSize()], payload[e.gcm.NonceSize():]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption: %v", err)
	}

	return plaintext, nil
}
