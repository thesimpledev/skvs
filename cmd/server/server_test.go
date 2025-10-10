//go:build !short

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

func TestServerIntegration(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &application{
		cfg: &config{port: 0},
		log: logger,
	}

	serverReady := make(chan int)
	serverErr := make(chan error)

	go func() {
		addr := ":0"
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			serverErr <- err
			return
		}

		server, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			serverErr <- err
			return
		}
		defer server.Close()

		actualPort := server.LocalAddr().(*net.UDPAddr).Port
		app.cfg.port = actualPort
		serverReady <- actualPort

		var bufPool = sync.Pool{
			New: func() any {
				buf := make([]byte, protocol.EncryptedFrameSize)
				return &buf
			},
		}

		for {
			bufPtr := bufPool.Get().(*[]byte)
			buf := *bufPtr
			n, clientAddr, err := server.ReadFromUDP(buf)
			if err != nil {
				bufPool.Put(bufPtr)
				continue
			}

			data := make([]byte, n)
			copy(data, buf[:n])
			bufPool.Put(bufPtr)

			go app.handlePacket(server, clientAddr, data)
		}
	}()

	var port int
	select {
	case port = <-serverReady:
		t.Logf("Server started on port %d", port)
	case err := <-serverErr:
		t.Fatalf("Server failed to start: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Server start timeout")
	}

	time.Sleep(100 * time.Millisecond)

	c, err := client.New(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("set and get", func(t *testing.T) {
		resp, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "testkey", "testvalue")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}
		if string(resp) != "testvalue" {
			t.Errorf("SET response: got %q, want %q", string(resp), "testvalue")
		}

		resp, err = c.Send(ctx, protocol.CMD_GET, 0, "testkey", "")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		if string(resp) != "testvalue" {
			t.Errorf("GET response: got %q, want %q", string(resp), "testvalue")
		}
	})

	t.Run("exists", func(t *testing.T) {
		_, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "existkey", "value")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}

		resp, err := c.Send(ctx, protocol.CMD_EXISTS, 0, "existkey", "")
		if err != nil {
			t.Fatalf("EXISTS failed: %v", err)
		}
		if string(resp) != "1" {
			t.Errorf("EXISTS response: got %q, want %q", string(resp), "1")
		}

		resp, err = c.Send(ctx, protocol.CMD_EXISTS, 0, "nonexistent", "")
		if err != nil {
			t.Fatalf("EXISTS failed: %v", err)
		}
		if string(resp) != "0" {
			t.Errorf("EXISTS response: got %q, want %q", string(resp), "0")
		}
	})

	t.Run("delete", func(t *testing.T) {
		_, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "delkey", "delvalue")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}

		resp, err := c.Send(ctx, protocol.CMD_DELETE, 0, "delkey", "")
		if err != nil {
			t.Fatalf("DELETE failed: %v", err)
		}
		if string(resp) != "delvalue" {
			t.Errorf("DELETE response: got %q, want %q", string(resp), "delvalue")
		}

		resp, err = c.Send(ctx, protocol.CMD_GET, 0, "delkey", "")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("GET after DELETE: expected empty, got %q", string(resp))
		}
	})

	t.Run("overwrite flag", func(t *testing.T) {
		_, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "overkey", "original")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}

		resp, err := c.Send(ctx, protocol.CMD_SET, 0, "overkey", "newvalue")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}
		if string(resp) != "original" {
			t.Errorf("SET without overwrite: got %q, want %q", string(resp), "original")
		}

		resp, err = c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "overkey", "newvalue")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}
		if string(resp) != "newvalue" {
			t.Errorf("SET with overwrite: got %q, want %q", string(resp), "newvalue")
		}

		resp, err = c.Send(ctx, protocol.CMD_GET, 0, "overkey", "")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		if string(resp) != "newvalue" {
			t.Errorf("GET after overwrite: got %q, want %q", string(resp), "newvalue")
		}
	})

	t.Run("old flag", func(t *testing.T) {
		_, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE, "oldkey", "original")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}

		resp, err := c.Send(ctx, protocol.CMD_SET, protocol.FLAG_OVERWRITE|protocol.FLAG_OLD, "oldkey", "newvalue")
		if err != nil {
			t.Fatalf("SET failed: %v", err)
		}
		if string(resp) != "original" {
			t.Errorf("SET with old flag: got %q, want %q", string(resp), "original")
		}

		resp, err = c.Send(ctx, protocol.CMD_GET, 0, "oldkey", "")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		if string(resp) != "newvalue" {
			t.Errorf("GET after old flag: got %q, want %q", string(resp), "newvalue")
		}
	})
}

func TestHandlePacketDecryptFailure(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &application{
		cfg: &config{port: 0},
		log: logger,
	}

	// Create a UDP server to receive the error response
	responseConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create response listener: %v", err)
	}
	defer responseConn.Close()

	responseAddr := responseConn.LocalAddr().(*net.UDPAddr)

	// Create a UDP connection for the server to write to
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create server conn: %v", err)
	}
	defer serverConn.Close()

	// Send invalid encrypted data
	invalidData := []byte("this is not properly encrypted data")

	done := make(chan bool)
	go func() {
		app.handlePacket(serverConn, responseAddr, invalidData)
		done <- true
	}()

	// Wait for response
	responseConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := responseConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected error response, got error: %v", err)
	}

	response := string(buf[:n])
	if response != "ERROR: failed to process message" {
		t.Errorf("Expected error message, got: %q", response)
	}

	<-done
}

func TestHandlePacketInvalidMessage(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &application{
		cfg: &config{port: 0},
		log: logger,
	}

	// Create encryption
	enc, err := encryption.New(nil)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Create a UDP server to receive the error response
	responseConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create response listener: %v", err)
	}
	defer responseConn.Close()

	responseAddr := responseConn.LocalAddr().(*net.UDPAddr)

	// Create a UDP connection for the server to write to
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create server conn: %v", err)
	}
	defer serverConn.Close()

	// Send properly encrypted but invalid protocol message (too short)
	invalidPayload := []byte{0xFF} // Invalid command, too short
	encryptedData, err := enc.Encrypt(invalidPayload)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	done := make(chan bool)
	go func() {
		app.handlePacket(serverConn, responseAddr, encryptedData)
		done <- true
	}()

	// Wait for response
	responseConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := responseConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected error response, got error: %v", err)
	}

	response := string(buf[:n])
	if response != "ERROR: failed to process message" {
		t.Errorf("Expected error message, got: %q", response)
	}

	<-done
}

func TestSendMessage(t *testing.T) {
	t.Setenv("SKVS_ENCRYPTION_KEY", "12345678901234567890123456789012")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := &application{
		cfg: &config{port: 0},
		log: logger,
	}

	// Create a UDP server to receive the message
	responseConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create response listener: %v", err)
	}
	defer responseConn.Close()

	responseAddr := responseConn.LocalAddr().(*net.UDPAddr)

	// Create a UDP connection for sending
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create server conn: %v", err)
	}
	defer serverConn.Close()

	testMessage := []byte("test message")
	app.sendMessage(testMessage, serverConn, responseAddr)

	// Wait for message
	responseConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := responseConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	if string(buf[:n]) != string(testMessage) {
		t.Errorf("Expected %q, got %q", testMessage, buf[:n])
	}
}
