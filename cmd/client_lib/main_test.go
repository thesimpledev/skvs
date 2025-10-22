package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type mockClient struct{}

func (m *mockClient) Send(ctx context.Context, command byte, flags uint32, key, value string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	resp := []byte(fmt.Sprintf("%d:%d:%s:%s", command, flags, key, value))
	return resp, nil
}

func (m *mockClient) Close() {}

func TestSetCallsSendWithExpectedArgs(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Set(ctx, "foo", "bar", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%d:%d:%s:%s", protocol.CMD_SET, protocol.FLAG_OVERWRITE, "foo", "bar")
	if got != expected {
		t.Errorf("unexpected send args: got %q, want %q", got, expected)
	}
}

func TestGetCallsSendCorrectly(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%d:%d:%s:%s", protocol.CMD_GET, 0, "foo", "")
	if got != expected {
		t.Errorf("unexpected send args: got %q, want %q", got, expected)
	}
}

func TestDeleteCallsSendCorrectly(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Delete(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%d:%d:%s:%s", protocol.CMD_DELETE, 0, "foo", "")
	if got != expected {
		t.Errorf("unexpected send args: got %q, want %q", got, expected)
	}
}

func TestExistsReturnsTrue(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Exists(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Errorf("expected true, got false")
	}
}

func TestExistsReturnsFalse(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Exists(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Errorf("expected false, got true")
	}
}

func TestSetWithOldFlag(t *testing.T) {
	mc := &mockClient{}
	s := newWithClient(mc)

	ctx := t.Context()

	got, err := s.Set(ctx, "key", "val", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFlags := protocol.FLAG_OVERWRITE | protocol.FLAG_OLD
	expected := fmt.Sprintf("%d:%d:%s:%s", protocol.CMD_SET, expectedFlags, "key", "val")
	if got != expected {
		t.Errorf("unexpected send args: got %q, want %q", got, expected)
	}
}
