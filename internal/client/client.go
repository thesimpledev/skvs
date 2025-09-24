package client

import (
	"fmt"
	"net"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type Client struct {
	addr *net.UDPAddr
	conn *net.UDPConn
}

func New(serverAddr string) (*Client, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	return &Client{addr: udpAddr, conn: conn}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) Send(command byte, flags uint32, key, value string) ([]byte, error) {
	frame := make([]byte, protocol.FrameSize)

	frame[0] = command

	frame[1] = byte(flags)
	frame[2] = byte(flags >> 8)
	frame[3] = byte(flags >> 16)
	frame[4] = byte(flags >> 24)

	copy(frame[5:5+protocol.KeySize], []byte(key))
	copy(frame[5+protocol.KeySize:], []byte(value))

	encrypted, err := encryption.Encrypt(frame)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	_, err = c.conn.Write(encrypted)
	if err != nil {
		return nil, fmt.Errorf("send frame: %w", err)
	}

	buf := make([]byte, protocol.EncryptedFrameSize)
	c.conn.SetReadDeadline(time.Now().Add(protocol.Timeout))
	n, _, err := c.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	decrypted, err := encryption.Decrypt(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return decrypted, nil

}
