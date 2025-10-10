package main

import (
	"bytes"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func buildFrame(cmd byte, flags uint32, key, value string) []byte {
	frame := make([]byte, protocol.FrameSize)
	frame[0] = cmd
	frame[1] = byte(flags)
	frame[2] = byte(flags >> 8)
	frame[3] = byte(flags >> 16)
	frame[4] = byte(flags >> 24)

	start := protocol.CommandSize + protocol.FlagSize
	copy(frame[start:start+protocol.KeySize], []byte(key))
	copy(frame[start+protocol.KeySize:], []byte(value))

	return frame
}

func TestProcessMessage(t *testing.T) {
	resetStorage()

	t.Run("invalid frame size", func(t *testing.T) {
		_, err := processMessage([]byte("too short"))
		if err == nil {
			t.Fatal("expected error for invalid frame size, got nil")
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		frame := buildFrame(99, 0, "key", "value")
		_, err := processMessage(frame)
		if err == nil {
			t.Fatal("expected error for unknown command, got nil")
		}
	})

	t.Run("CMD_SET", func(t *testing.T) {
		resetStorage()
		frame := buildFrame(protocol.CMD_SET, protocol.FLAG_OVERWRITE, "testkey", "testvalue")

		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("testvalue")) {
			t.Errorf("expected %q, got %q", "testvalue", string(resp))
		}
		if !bytes.Equal(skvs["testkey"], []byte("testvalue")) {
			t.Error("value not stored")
		}
	})

	t.Run("CMD_GET", func(t *testing.T) {
		resetStorage()
		skvs["getkey"] = []byte("getvalue")

		frame := buildFrame(protocol.CMD_GET, 0, "getkey", "")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("getvalue")) {
			t.Errorf("expected %q, got %q", "getvalue", string(resp))
		}
	})

	t.Run("CMD_DELETE", func(t *testing.T) {
		resetStorage()
		skvs["delkey"] = []byte("delvalue")

		frame := buildFrame(protocol.CMD_DELETE, 0, "delkey", "")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("delvalue")) {
			t.Errorf("expected %q, got %q", "delvalue", string(resp))
		}
		if _, exists := skvs["delkey"]; exists {
			t.Error("key should be deleted")
		}
	})

	t.Run("CMD_EXISTS true", func(t *testing.T) {
		resetStorage()
		skvs["existkey"] = []byte("value")

		frame := buildFrame(protocol.CMD_EXISTS, 0, "existkey", "")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("1")) {
			t.Errorf("expected '1', got %q", string(resp))
		}
	})

	t.Run("CMD_EXISTS false", func(t *testing.T) {
		resetStorage()

		frame := buildFrame(protocol.CMD_EXISTS, 0, "nonexistent", "")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("0")) {
			t.Errorf("expected '0', got %q", string(resp))
		}
	})

	t.Run("flag extraction - overwrite", func(t *testing.T) {
		resetStorage()
		skvs["key"] = []byte("original")

		frame := buildFrame(protocol.CMD_SET, protocol.FLAG_OVERWRITE, "key", "new")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("new")) {
			t.Errorf("expected overwrite to work")
		}
		if !bytes.Equal(skvs["key"], []byte("new")) {
			t.Error("value should be overwritten")
		}
	})

	t.Run("flag extraction - old", func(t *testing.T) {
		resetStorage()
		skvs["key"] = []byte("original")

		frame := buildFrame(protocol.CMD_SET, protocol.FLAG_OVERWRITE|protocol.FLAG_OLD, "key", "new")
		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("original")) {
			t.Errorf("expected old value, got %q", string(resp))
		}
		if !bytes.Equal(skvs["key"], []byte("new")) {
			t.Error("value should be updated")
		}
	})

	t.Run("null-padded key and value", func(t *testing.T) {
		resetStorage()
		frame := make([]byte, protocol.FrameSize)
		frame[0] = protocol.CMD_SET
		frame[1] = byte(protocol.FLAG_OVERWRITE)

		start := protocol.CommandSize + protocol.FlagSize
		copy(frame[start:], []byte("k\x00\x00\x00"))
		copy(frame[start+protocol.KeySize:], []byte("v\x00\x00\x00"))

		resp, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(resp, []byte("v")) {
			t.Errorf("expected trimmed value")
		}
		if !bytes.Equal(skvs["k"], []byte("v")) {
			t.Error("key/value should be trimmed")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		resetStorage()
		frame := buildFrame(protocol.CMD_SET, protocol.FLAG_OVERWRITE, "", "value")

		_, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(skvs[""], []byte("value")) {
			t.Error("empty key should be allowed")
		}
	})

	t.Run("empty value", func(t *testing.T) {
		resetStorage()
		frame := buildFrame(protocol.CMD_SET, protocol.FLAG_OVERWRITE, "key", "")

		_, err := processMessage(frame)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(skvs["key"], []byte{}) {
			t.Error("empty value should be allowed")
		}
	})
}
