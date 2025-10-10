package main

import (
	"bytes"
	"testing"
)

func resetStorage() {
	mu.Lock()
	defer mu.Unlock()
	skvs = make(map[string][]byte)
}

func TestSet(t *testing.T) {
	t.Run("set new key", func(t *testing.T) {
		resetStorage()

		result, err := set("key1", []byte("value1"), false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("value1")) {
			t.Errorf("expected %q, got %q", "value1", string(result))
		}
		if !bytes.Equal(skvs["key1"], []byte("value1")) {
			t.Error("key not stored in map")
		}
	})

	t.Run("set existing key without overwrite", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("original")

		result, err := set("key1", []byte("newvalue"), false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("original")) {
			t.Errorf("expected original value, got %q", string(result))
		}
		if !bytes.Equal(skvs["key1"], []byte("original")) {
			t.Error("value should not have changed")
		}
	})

	t.Run("set existing key with overwrite", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("original")

		result, err := set("key1", []byte("newvalue"), true, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("newvalue")) {
			t.Errorf("expected new value, got %q", string(result))
		}
		if !bytes.Equal(skvs["key1"], []byte("newvalue")) {
			t.Error("value should have changed")
		}
	})

	t.Run("set with old flag returns previous value", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("original")

		result, err := set("key1", []byte("newvalue"), true, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("original")) {
			t.Errorf("expected original value, got %q", string(result))
		}
		if !bytes.Equal(skvs["key1"], []byte("newvalue")) {
			t.Error("value should have changed")
		}
	})

	t.Run("set new key with old flag returns nil", func(t *testing.T) {
		resetStorage()

		result, err := set("newkey", []byte("value"), false, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existent key with old flag, got %q", string(result))
		}
		if !bytes.Equal(skvs["newkey"], []byte("value")) {
			t.Error("value should have been set")
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("get existing key", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("value1")

		result, err := get("key1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("value1")) {
			t.Errorf("expected %q, got %q", "value1", string(result))
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		resetStorage()

		result, err := get("nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existent key, got %q", string(result))
		}
	})
}

func TestDel(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("value1")

		result, err := del("key1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("value1")) {
			t.Errorf("expected deleted value, got %q", string(result))
		}
		if _, exists := skvs["key1"]; exists {
			t.Error("key should have been deleted")
		}
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		resetStorage()

		result, err := del("nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existent key, got %q", string(result))
		}
	})
}

func TestExists(t *testing.T) {
	t.Run("key exists", func(t *testing.T) {
		resetStorage()
		skvs["key1"] = []byte("value1")

		result, err := exists("key1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("1")) {
			t.Errorf("expected '1', got %q", string(result))
		}
	})

	t.Run("key does not exist", func(t *testing.T) {
		resetStorage()

		result, err := exists("nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, []byte("0")) {
			t.Errorf("expected '0', got %q", string(result))
		}
	})
}

func TestConcurrency(t *testing.T) {
	resetStorage()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(n int) {
			key := "key"
			value := []byte("value")
			set(key, value, true, false)
			get(key)
			del(key)
			exists(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
