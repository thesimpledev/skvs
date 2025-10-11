package client

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

func TestNew(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	t.Run("valid address", func(t *testing.T) {
		c, err := New("localhost:4040", validKey)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c == nil {
			t.Fatal("expected client, got nil")
		}
		defer c.Close()
	})

	t.Run("invalid address format", func(t *testing.T) {
		_, err := New("invalid::address", validKey)
		if err == nil {
			t.Fatal("expected error for invalid address, got nil")
		}
	})

	t.Run("invalid encryption key", func(t *testing.T) {
		_, err := New("localhost:4040", []byte("tooshort"))
		if err == nil {
			t.Fatal("expected error for invalid encryption key, got nil")
		}
	})
}

func TestClose(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	c, err := New("localhost:4040", validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.Send(ctx, protocol.CMD_GET, 0, "test", "")
	if err == nil {
		t.Fatal("expected error after close, got nil")
	}
}

func TestSendRequiresDeadline(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	c, err := New("localhost:4040", validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	_, err = c.Send(ctx, protocol.CMD_GET, 0, "test", "")
	if err == nil {
		t.Fatal("expected error for context without deadline, got nil")
	}
	if err.Error() != "Send requires a context with deadline" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSendFrameBuilding(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	testCases := []struct {
		name    string
		command byte
		flags   uint32
		key     string
		value   string
	}{
		{"SET command", protocol.CMD_SET, 0, "mykey", "myvalue"},
		{"GET command", protocol.CMD_GET, 0, "testkey", ""},
		{"with overwrite flag", protocol.CMD_SET, protocol.FLAG_OVERWRITE, "key", "val"},
		{"with old flag", protocol.CMD_SET, protocol.FLAG_OLD, "key", "val"},
		{"multiple flags", protocol.CMD_SET, protocol.FLAG_OVERWRITE | protocol.FLAG_OLD, "k", "v"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			frame := make([]byte, protocol.FrameSize)
			frame[0] = tc.command
			frame[1] = byte(tc.flags)
			frame[2] = byte(tc.flags >> 8)
			frame[3] = byte(tc.flags >> 16)
			frame[4] = byte(tc.flags >> 24)
			copy(frame[5:5+protocol.KeySize], []byte(tc.key))
			copy(frame[5+protocol.KeySize:], []byte(tc.value))

			if frame[0] != tc.command {
				t.Errorf("command mismatch: got %d, want %d", frame[0], tc.command)
			}

			reconstructedFlags := uint32(frame[1]) |
				uint32(frame[2])<<8 |
				uint32(frame[3])<<16 |
				uint32(frame[4])<<24
			if reconstructedFlags != tc.flags {
				t.Errorf("flags mismatch: got %d, want %d", reconstructedFlags, tc.flags)
			}

			keyBytes := frame[5 : 5+protocol.KeySize]
			keyStr := string(keyBytes[:len(tc.key)])
			if keyStr != tc.key {
				t.Errorf("key mismatch: got %q, want %q", keyStr, tc.key)
			}

			valBytes := frame[5+protocol.KeySize : 5+protocol.KeySize+protocol.ValueSize]
			if tc.value != "" {
				valStr := string(valBytes[:len(tc.value)])
				if valStr != tc.value {
					t.Errorf("value mismatch: got %q, want %q", valStr, tc.value)
				}
			}
		})
	}
}

func TestSendEncryptionError(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	c, err := New("localhost:4040", validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.Send(ctx, protocol.CMD_GET, 0, "test", "")
	if err == nil {
		t.Fatal("expected error (no server), got nil")
	}
}

func TestSendWithMockServer(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0") // :0 = random port
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer serverConn.Close()

	actualPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	enc, err := encryption.New(validKey)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	serverDone := make(chan bool)
	go func() {
		defer close(serverDone)
		buf := make([]byte, protocol.EncryptedFrameSize)
		n, clientAddr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		_, err = enc.Decrypt(buf[:n])
		if err != nil {
			t.Logf("server decrypt error: %v", err)
			return
		}

		response := []byte("OK")
		encrypted, err := enc.Encrypt(response)
		if err != nil {
			t.Logf("server encrypt error: %v", err)
			return
		}

		serverConn.WriteToUDP(encrypted, clientAddr)
	}()

	client, err := New(fmt.Sprintf("127.0.0.1:%d", actualPort), validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Send(ctx, protocol.CMD_GET, 0, "testkey", "")
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if string(resp) != "OK" {
		t.Errorf("unexpected response: got %q, want %q", string(resp), "OK")
	}

	<-serverDone
}

func TestSendContextCancellation(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	client, err := New("127.0.0.1:9999", validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = client.Send(ctx, protocol.CMD_GET, 0, "test", "")
	if err == nil {
		t.Fatal("expected error due to timeout, got nil")
	}
}

func TestSendRetry(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012")

	client, err := New("127.0.0.1:9998", validKey)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = client.Send(ctx, protocol.CMD_GET, 0, "test", "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected retries to take time, got %v", elapsed)
	}
}
