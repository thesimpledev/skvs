package main

import (
	"context"
	"fmt"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type Request struct {
	Command   byte
	Key       string
	Value     string
	Overwrite bool
	Old       bool
}

type Client interface {
	Close()
	Send(ctx context.Context, command byte, flags uint32, key, value string) ([]byte, error)
}

type Skvs struct {
	client Client
}

func New(addr string) (*Skvs, error) {
	c, err := client.New(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return &Skvs{c}, nil
}

func NewWithClient(c Client) *Skvs {
	return &Skvs{client: c}
}

func (s *Skvs) do(ctx context.Context, req Request) (string, error) {
	var flags uint32
	if req.Overwrite {
		flags |= protocol.FLAG_OVERWRITE
	}
	if req.Old {
		flags |= protocol.FLAG_OLD
	}

	resp, err := s.client.Send(ctx, req.Command, flags, req.Key, req.Value)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (s *Skvs) Set(ctx context.Context, key, value string, overwrite, old bool) (string, error) {
	return s.do(ctx, Request{Command: protocol.CMD_SET, Key: key, Value: value, Overwrite: overwrite, Old: old})
}

func (s *Skvs) Get(ctx context.Context, key string) (string, error) {
	return s.do(ctx, Request{Command: protocol.CMD_GET, Key: key})
}

func (s *Skvs) Delete(ctx context.Context, key string) (string, error) {
	return s.do(ctx, Request{Command: protocol.CMD_DELETE, Key: key})
}

func (s *Skvs) Exists(ctx context.Context, key string) (bool, error) {
	resp, err := s.do(ctx, Request{Command: protocol.CMD_EXISTS, Key: key})
	if err != nil {
		return false, err
	}
	return resp == "1", nil
}

func (s *Skvs) Close() {
	s.client.Close()
}
