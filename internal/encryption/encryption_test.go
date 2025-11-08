package encryption

import (
	"bytes"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
		err  bool
	}{
		{
			name: "nil key",
			key:  nil,
			err:  true,
		},
		{
			name: "invalid key length",
			key:  []byte("key"),
			err:  true,
		},
		{
			name: "valid key",
			key:  []byte("asdfhjshajshehdhdkfhehdhsakjhhki"),
			err:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := New(tt.key)

			if err != nil && !tt.err {
				t.Fatalf("error got error expected no error %v", err)
			}

			if tt.err {
				return
			}

			if encryptor == nil {
				t.Errorf("expected encryptor got nil")
			}
		})
	}
}

func TestEncryptionDecryption(t *testing.T) {
	encryptor, err := New([]byte("asdfhjshajshehdhdkfhehdhsakjhhki"))
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	dto := protocol.FrameDTO{
		Cmd:   protocol.CMD_SET,
		Key:   "testkey",
		Value: []byte("Hello world this is really secret"),
	}
	expected := protocol.DtoToFrame(dto)

	payload, err := encryptor.Encrypt(expected)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	got, err := encryptor.Decrypt(payload)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if !bytes.Equal(expected, got) {
		t.Errorf("expected %v got %v", expected, got)
	}
}

func TestEncryptionDecryptionWrongLength(t *testing.T) {
	encryptor, err := New([]byte("asdfhjshajshehdhdkfhehdhsakjhhki"))
	expected := []byte("Hello world this is really secret")
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	_, err = encryptor.Encrypt(expected)
	if err == nil {
		t.Errorf("Expected Length error got none")
	}
}

func TestDecryptShortPayload(t *testing.T) {
	encryptor, _ := New([]byte("asdfhjshajshehdhdkfhehdhsakjhhki"))

	shortPayload := []byte("short")
	_, err := encryptor.Decrypt(shortPayload)

	if err == nil {
		t.Error("expected error for short payload, got nil")
	}
}
