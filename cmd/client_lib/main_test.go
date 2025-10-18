package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

func TestNew(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	c, err := New("localhost:4040")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected client, got nil")
	}
	defer c.Close()
}

// Mock server helper
type mockServerInfo struct {
	addr string
	done chan bool
}

func (m *mockServerInfo) Addr() string {
	return m.addr
}

func (m *mockServerInfo) Close() {
	close(m.done)
}

func TestDoWithMockServer(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	// We need a more sophisticated mock that can verify what was sent
	mockServer := startVerifyingMockServer(t)
	defer mockServer.Close()

	c, err := New(mockServer.Addr())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Run("Do with SET command", func(t *testing.T) {
		_, err := c.Do(ctx, Request{
			Command:   CommandSet,
			Key:       "testkey",
			Value:     "testvalue",
			Overwrite: true,
			Old:       false,
		})
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		// Verify mock received CMD_SET with FLAG_OVERWRITE
		if mockServer.lastCmd != protocol.CMD_SET {
			t.Errorf("expected CMD_SET (%d), got %d", protocol.CMD_SET, mockServer.lastCmd)
		}
		if mockServer.lastFlags != protocol.FLAG_OVERWRITE {
			t.Errorf("expected FLAG_OVERWRITE (%d), got %d", protocol.FLAG_OVERWRITE, mockServer.lastFlags)
		}
	})

	t.Run("Do with GET command", func(t *testing.T) {
		_, err := c.Do(ctx, Request{Command: CommandGet, Key: "key"})
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		if mockServer.lastCmd != protocol.CMD_GET {
			t.Errorf("expected CMD_GET (%d), got %d", protocol.CMD_GET, mockServer.lastCmd)
		}
	})

	t.Run("Do with DELETE command", func(t *testing.T) {
		_, err := c.Do(ctx, Request{Command: CommandDelete, Key: "key"})
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		if mockServer.lastCmd != protocol.CMD_DELETE {
			t.Errorf("expected CMD_DELETE (%d), got %d", protocol.CMD_DELETE, mockServer.lastCmd)
		}
	})

	t.Run("Do with EXISTS command", func(t *testing.T) {
		_, err := c.Do(ctx, Request{Command: CommandExists, Key: "key"})
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		if mockServer.lastCmd != protocol.CMD_EXISTS {
			t.Errorf("expected CMD_EXISTS (%d), got %d", protocol.CMD_EXISTS, mockServer.lastCmd)
		}
	})

	t.Run("Do with both flags", func(t *testing.T) {
		_, err := c.Do(ctx, Request{
			Command:   CommandSet,
			Key:       "key",
			Value:     "val",
			Overwrite: true,
			Old:       true,
		})
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		expectedFlags := protocol.FLAG_OVERWRITE | protocol.FLAG_OLD
		if mockServer.lastFlags != expectedFlags {
			t.Errorf("expected flags %d, got %d", expectedFlags, mockServer.lastFlags)
		}
	})

	t.Run("Do with unknown command", func(t *testing.T) {
		_, err := c.Do(ctx, Request{Command: "UNKNOWN", Key: "key"})
		if err == nil {
			t.Fatal("expected error for unknown command")
		}
	})
}

func TestExistsMethod(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	t.Run("returns true for '1'", func(t *testing.T) {
		mockServer := startMockServerWithResponse(t, []byte("1"))
		defer mockServer.Close()

		c, _ := New(mockServer.Addr())
		defer c.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		exists, err := c.Exists(ctx, "key")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("expected true, got false")
		}
	})

	t.Run("returns false for '0'", func(t *testing.T) {
		mockServer := startMockServerWithResponse(t, []byte("0"))
		defer mockServer.Close()

		c, _ := New(mockServer.Addr())
		defer c.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		exists, err := c.Exists(ctx, "key")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if exists {
			t.Error("expected false, got true")
		}
	})
}

// Enhanced mock server that tracks what was received
type verifyingMockServer struct {
	addr      string
	done      chan bool
	lastCmd   byte
	lastFlags uint32
	enc       *encryption.Encryptor
}

func (m *verifyingMockServer) Addr() string { return m.addr }
func (m *verifyingMockServer) Close()       { close(m.done) }

func startVerifyingMockServer(t *testing.T) *verifyingMockServer {
	t.Helper()

	enc, _ := encryption.New(nil)

	serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	serverConn, _ := net.ListenUDP("udp", serverAddr)
	actualPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	mock := &verifyingMockServer{
		addr: fmt.Sprintf("127.0.0.1:%d", actualPort),
		done: make(chan bool),
		enc:  enc,
	}

	go func() {
		defer func() {
			_ = serverConn.Close()
		}()
		for {
			select {
			case <-mock.done:
				return
			default:
				buf := make([]byte, protocol.EncryptedFrameSize)
				err := serverConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				if err != nil {
					continue
				}
				n, clientAddr, err := serverConn.ReadFromUDP(buf)
				if err != nil {
					continue
				}

				decrypted, err := enc.Decrypt(buf[:n])
				if err != nil {
					continue
				}

				// Parse the frame to extract cmd and flags
				if len(decrypted) >= 5 {
					mock.lastCmd = decrypted[0]
					mock.lastFlags = uint32(decrypted[1]) |
						uint32(decrypted[2])<<8 |
						uint32(decrypted[3])<<16 |
						uint32(decrypted[4])<<24
				}

				response := []byte("OK")
				encrypted, err := enc.Encrypt(response)
				if err != nil {
					continue
				}
				_, err = serverConn.WriteToUDP(encrypted, clientAddr)
				if err != nil {
					continue
				}
			}
		}
	}()

	return mock
}

func startMockServerWithResponse(t *testing.T, response []byte) *mockServerInfo {
	t.Helper()

	enc, _ := encryption.New(nil)
	serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	serverConn, _ := net.ListenUDP("udp", serverAddr)
	actualPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	done := make(chan bool)

	go func() {
		defer func() {
			_ = serverConn.Close()
		}()
		for {
			select {
			case <-done:
				return
			default:
				buf := make([]byte, protocol.EncryptedFrameSize)
				err := serverConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				if err != nil {
					continue
				}
				n, clientAddr, err := serverConn.ReadFromUDP(buf)
				if err != nil {
					continue
				}

				_, err = enc.Decrypt(buf[:n])
				if err != nil {
					continue
				}

				encrypted, err := enc.Encrypt(response)
				if err != nil {
					continue
				}
				_, err = serverConn.WriteToUDP(encrypted, clientAddr)
				if err != nil {
					continue
				}
			}
		}
	}()

	return &mockServerInfo{
		addr: fmt.Sprintf("127.0.0.1:%d", actualPort),
		done: done,
	}
}

func TestSetGetDeleteErrorPaths(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	// Create client pointing to non-existent server to trigger errors
	c, err := New("127.0.0.1:9998")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	t.Run("Set returns error when Send fails", func(t *testing.T) {
		_, err := c.Set(ctx, "key", "value", true, false)
		if err == nil {
			t.Fatal("expected error from Set, got nil")
		}
	})

	t.Run("Get returns error when Send fails", func(t *testing.T) {
		_, err := c.Get(ctx, "key")
		if err == nil {
			t.Fatal("expected error from Get, got nil")
		}
	})

	t.Run("Delete returns error when Send fails", func(t *testing.T) {
		_, err := c.Delete(ctx, "key")
		if err == nil {
			t.Fatal("expected error from Delete, got nil")
		}
	})

	t.Run("Exists returns error when Send fails", func(t *testing.T) {
		_, err := c.Exists(ctx, "key")
		if err == nil {
			t.Fatal("expected error from Exists, got nil")
		}
	})
}

func TestNewErrorPath(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	t.Run("New returns error for invalid address", func(t *testing.T) {
		_, err := New("invalid::address::format")
		if err == nil {
			t.Fatal("expected error for invalid address, got nil")
		}
		// Verify the error message contains our wrapper text
		if !strings.Contains(err.Error(), "create client") {
			t.Errorf("expected error to contain 'create client', got: %v", err)
		}
	})
}
