package main

import (
	"context"
	"fmt"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

// Public command constants
const (
	CommandSet    = "SET"
	CommandGet    = "GET"
	CommandDelete = "DELETE"
	CommandExists = "EXISTS"
)

// Request models a single operation against the server.
type Request struct {
	Command   string
	Key       string
	Value     string
	Overwrite bool
	Old       bool
}

// Client is the external-facing SKVS client.
type Client struct {
	*client.Client
}

// New creates a new client connected to the given addr.
func New(addr string) (*Client, error) {
	c, err := client.New(addr)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &Client{c}, nil
}

// Do executes a Request.
func (c *Client) Do(ctx context.Context, req Request) (string, error) {
	var cmd byte
	switch req.Command {
	case CommandSet:
		cmd = protocol.CMD_SET
	case CommandGet:
		cmd = protocol.CMD_GET
	case CommandDelete:
		cmd = protocol.CMD_DELETE
	case CommandExists:
		cmd = protocol.CMD_EXISTS
	default:
		return "", fmt.Errorf("unknown command: %s", req.Command)
	}

	var flags uint32
	if req.Overwrite {
		flags |= protocol.FLAG_OVERWRITE
	}
	if req.Old {
		flags |= protocol.FLAG_OLD
	}

	resp, err := c.Send(ctx, cmd, flags, req.Key, req.Value)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// Convenience helpers ---------------------------------------------------------

func (c *Client) Set(ctx context.Context, key, value string, overwrite, old bool) (string, error) {
	return c.Do(ctx, Request{Command: CommandSet, Key: key, Value: value, Overwrite: overwrite, Old: old})
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Do(ctx, Request{Command: CommandGet, Key: key})
}

func (c *Client) Delete(ctx context.Context, key string) (string, error) {
	return c.Do(ctx, Request{Command: CommandDelete, Key: key})
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	resp, err := c.Do(ctx, Request{Command: CommandExists, Key: key})
	if err != nil {
		return false, err
	}
	return resp == "1", nil
}
