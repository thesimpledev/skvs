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
	encryptor *encryption.Encryptor
}

func New(serverAddr string, encryptionKey []byte) (*Client, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	e, err := encryption.New(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &Client{addr: udpAddr, encryptor: e}, nil
}

func (c *Client) Close() {}

func (c *Client) roundTrip(encrypted []byte, writeDeadline, readDeadline time.Time) ([]byte, error) {
	conn, err := net.DialUDP("udp", nil, c.addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}
	defer func() { _ = conn.Close() }()

	err = conn.SetWriteDeadline(writeDeadline)
	if err != nil {
		return nil, fmt.Errorf("set write deadline: %w", err)
	}

	err = conn.SetReadDeadline(readDeadline)
	if err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}

	_, err = conn.Write(encrypted)
	if err != nil {
		return nil, fmt.Errorf("send frame: %w", err)
	}

	buf := make([]byte, protocol.EncryptedFrameSize)

	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return buf[:n], nil
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

	var lastError error
	for attempt := range maxAttempts {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		backoffErr := c.backoff(ctx, attempt)
		if backoffErr != nil {
			return "", backoffErr
		}

		responseDTO, attemptErr := c.attemptOnce(encrypted, deadline, attempt)
		if attemptErr != nil {
			lastError = attemptErr
			continue
		}

		if responseDTO.Status == protocol.STATUS_ERROR {
			return "", fmt.Errorf("server error: %s", string(responseDTO.Value))
		}

		return string(responseDTO.Value), nil
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastError)
}

func (c *Client) backoff(ctx context.Context, attempt int) error {
	if attempt == 0 {
		return nil
	}

	delay := min(baseDelay*(1<<(attempt-1)), time.Second)
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) attemptOnce(encrypted []byte, deadline time.Time, attempt int) (protocol.ResponseDTO, error) {
	readDeadline := time.Now().Add(min(baseDelay*(1<<attempt), time.Second))
	if readDeadline.After(deadline) {
		readDeadline = deadline
	}

	response, err := c.roundTrip(encrypted, deadline, readDeadline)
	if err != nil {
		return protocol.ResponseDTO{}, err
	}

	decrypted, err := c.encryptor.Decrypt(response)
	if err != nil {
		return protocol.ResponseDTO{}, fmt.Errorf("decryption failed: %w", err)
	}

	responseDTO, err := protocol.FrameToResponseDTO(decrypted)
	if err != nil {
		return protocol.ResponseDTO{}, fmt.Errorf("parse response failed: %w", err)
	}

	return responseDTO, nil
}
