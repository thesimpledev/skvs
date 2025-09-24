package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
)

var key = []byte(os.Getenv("SKVS_ENCRYPTION_KEY"))

func init() {
	if len(key) != 32 {
		log.Fatal("SKVS_ENCRYPTION_KEY must be exactly 32 bytes for AES-256-GCM")
	}

}

func Encrypt(payload []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
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

func Decrypt(payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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
