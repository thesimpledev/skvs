//go:build exclude_tests

package main

import (
	"context"
	"fmt"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type clientLibrary struct {
	client *client.Client
}

func New(addr string) (*clientLibrary, error) {
	c, err := client.New(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &clientLibrary{client: c}, nil
}

func (c *clientLibrary) Set(ctx context.Context, key, value string, overwrite, old bool) (string, error) {
	dto, err := protocol.NewFrameDTO("set", key, value, overwrite, old)
	if err != nil {
		return "", fmt.Errorf("set failed for key: %s - value: %s with error %v", key, value, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *clientLibrary) Get(ctx context.Context, key string) (string, error) {
	dto, err := protocol.NewFrameDTO("get", key, "", false, false)
	if err != nil {
		return "", fmt.Errorf("get failed for key: %s with error %v", key, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *clientLibrary) Delete(ctx context.Context, key string) (string, error) {
	dto, err := protocol.NewFrameDTO("delete", key, "", false, false)
	if err != nil {
		return "", fmt.Errorf("delete failed for key: %s with error %v", key, err)
	}

	return c.client.Send(ctx, dto)
}

func (c *clientLibrary) Exists(ctx context.Context, key string) (bool, error) {
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

func (c *clientLibrary) Close() {
	c.client.Close()
}
