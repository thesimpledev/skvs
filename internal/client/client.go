// Package client provides a client for the skvs service.
package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

const (
	maxAttempts = 10
	baseDelay   = 100 * time.Millisecond
)

type Client struct {
	addr      *net.UDPAddr
	conn      *net.UDPConn
	encryptor *encryption.Encryptor
}

func New(serverAddr string, encryptionKey []byte) (*Client, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	e, err := encryption.New(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &Client{addr: udpAddr, conn: conn, encryptor: e}, nil
}

func (c *Client) Close() {
	_ = c.conn.Close()
}

func (c *Client) Send(ctx context.Context, dto protocol.FrameDTO) (string, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return "", fmt.Errorf("Send requires a context with deadline")
	}

	frame := protocol.DtoToFrame(dto)

	encrypted, err := c.encryptor.Encrypt(frame)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}
	if len(encrypted) != protocol.EncryptedFrameSize {
		return "", fmt.Errorf("encrypted frame size mismatch: got %d, want %d", len(encrypted), protocol.EncryptedFrameSize)
	}

	var lastError error
	for attempt := range maxAttempts {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if attempt > 0 {
			delay := min(baseDelay*(1<<(attempt-1)), time.Second)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		_ = c.conn.SetWriteDeadline(deadline)
		_ = c.conn.SetReadDeadline(deadline)

		_, err = c.conn.Write(encrypted)
		if err != nil {
			lastError = fmt.Errorf("send frame: %w", err)
			continue
		}

		buf := make([]byte, protocol.EncryptedFrameSize)

		n, err := c.conn.Read(buf)
		if err != nil {
			lastError = fmt.Errorf("read response: %w", err)
			continue
		}

		decrypted, err := c.encryptor.Decrypt(buf[:n])
		if err != nil {
			lastError = fmt.Errorf("decryption failed: %w", err)
			continue
		}

		return string(decrypted), nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastError)
}
