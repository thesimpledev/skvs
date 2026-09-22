// Package clientlibrary provides a client library for the skvs server.
package clientlibrary

import (
	"context"
	"fmt"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

// Client talks to an skvs server. Create one with New.
type Client struct {
	client *client.Client
}

// New connects to the skvs server at addr using the given AES-256 key.
func New(addr string, key []byte) (*Client, error) {
	c, err := client.New(addr, key)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &Client{client: c}, nil
}

func (c *Client) Set(ctx context.Context, key, value string, overwrite, old bool) (string, error) {
	dto, err := protocol.NewFrameDTO("set", key, value, overwrite, old)
	if err != nil {
		return "", fmt.Errorf("set failed for key: %s - value: %s with error %v", key, value, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	dto, err := protocol.NewFrameDTO("get", key, "", false, false)
	if err != nil {
		return "", fmt.Errorf("get failed for key: %s with error %v", key, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *Client) Delete(ctx context.Context, key string) (string, error) {
	dto, err := protocol.NewFrameDTO("delete", key, "", false, false)
	if err != nil {
		return "", fmt.Errorf("delete failed for key: %s with error %v", key, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	dto, err := protocol.NewFrameDTO("exists", key, "", false, false)
	if err != nil {
		return false, fmt.Errorf("exists failed for key: %s with error %v", key, err)
	}

	resp, err := c.client.Send(ctx, dto)
	if err != nil {
		return false, err
	}
	return resp == "1", nil
}

func (c *Client) Close() {
	c.client.Close()
}
