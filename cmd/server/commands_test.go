package main

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
)

type te struct{}

func (e *te) Encrypt(input []byte) ([]byte, error) {
	return input, nil
}

func (e *te) Decrypt(input []byte) ([]byte, error) {
	return input, nil
}

func newTestApp() *application {
	config := &config{
		port: 8080,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	e := &te{}

	app := &application{
		cfg:       config,
		log:       logger,
		encryptor: e,
		skvs:      make(map[string][]byte),
	}

	return app
}

func TestSet(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		initial    []byte
		value      []byte
		overwrite  bool
		old        bool
		wantMap    []byte
		wantReturn []byte
		wantErr    bool
	}{
		{
			name:       "set new key",
			key:        "foo",
			initial:    nil,
			value:      []byte("bar"),
			overwrite:  false,
			old:        false,
			wantMap:    []byte("bar"),
			wantReturn: []byte("bar"),
		},
		{
			name:       "overwrite existing key return new key",
			key:        "foo",
			initial:    []byte("initial"),
			value:      []byte("bar"),
			overwrite:  true,
			old:        false,
			wantMap:    []byte("bar"),
			wantReturn: []byte("bar"),
		},
		{
			name:       "over write existing key return old key",
			key:        "foo",
			initial:    []byte("initial"),
			value:      []byte("bar"),
			overwrite:  true,
			old:        true,
			wantMap:    []byte("bar"),
			wantReturn: []byte("initial"),
		},
		{
			name:       "do not overwrite, return old value",
			key:        "foo",
			initial:    []byte("initial"),
			value:      []byte("bar"),
			overwrite:  false,
			old:        true,
			wantMap:    []byte("initial"),
			wantReturn: []byte("initial"),
		},
		{
			name:       "overwrite nil key return new key",
			key:        "foo",
			initial:    nil,
			value:      []byte("bar"),
			overwrite:  true,
			old:        false,
			wantMap:    []byte("bar"),
			wantReturn: []byte("bar"),
		},
		{
			name:       "over write nil key return old key",
			key:        "foo",
			initial:    nil,
			value:      []byte("bar"),
			overwrite:  true,
			old:        true,
			wantMap:    []byte("bar"),
			wantReturn: []byte(""),
		},
		{
			name:       "do not overwrite nil key, return old value",
			key:        "foo",
			initial:    nil,
			value:      []byte("bar"),
			overwrite:  false,
			old:        true,
			wantMap:    []byte("bar"),
			wantReturn: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp()
			if tt.initial != nil {
				_, _ = app.set(tt.key, tt.initial, false, false)
			}
			got, err := app.set(tt.key, tt.value, tt.overwrite, tt.old)
			if (err != nil) != tt.wantErr {
				t.Errorf("set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if string(got) != string(tt.wantReturn) {
				t.Errorf("set() = %v, want %v", string(got), string(tt.wantReturn))
			}

			if string(app.skvs[tt.key]) != string(tt.wantMap) {
				t.Errorf("sksv = %v, want %v", string(app.skvs[tt.key]), string(tt.wantMap))
			}
		})
	}
}

func TestGet(t *testing.T) {
	app := newTestApp()

	want := []byte("Jack")

	_, err := app.set("cat", want, false, false)
	if err != nil {
		t.Fatal("Get Test failed during set")
	}

	got := app.get("cat")

	if !bytes.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}
