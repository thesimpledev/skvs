package skvs

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type testApp struct{}

func (app *testApp) set(_ string, _ []byte, _, _ bool) protocol.ResponseDTO {
	return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("set"))
}

func (app *testApp) get(_ string) protocol.ResponseDTO {
	return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("get"))
}

func (app *testApp) del(_ string) protocol.ResponseDTO {
	return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("del"))
}

func (app *testApp) exists(_ string) protocol.ResponseDTO {
	return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("exists"))
}

func newTestApp() *App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := &App{
		log:  logger,
		skvs: make(map[string][]byte),
	}

	return app
}

func TestCommandrouting(t *testing.T) {
	tests := []struct {
		name       string
		frame      protocol.FrameDTO
		wantValue  []byte
		wantStatus byte
	}{
		{
			name: "set command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_SET,
			},
			wantValue:  []byte("set"),
			wantStatus: protocol.STATUS_OK,
		},
		{
			name: "get command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_GET,
			},
			wantValue:  []byte("get"),
			wantStatus: protocol.STATUS_OK,
		},
		{
			name: "del command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_DELETE,
			},
			wantValue:  []byte("del"),
			wantStatus: protocol.STATUS_OK,
		},
		{
			name: "exists command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_EXISTS,
			},
			wantValue:  []byte("exists"),
			wantStatus: protocol.STATUS_OK,
		},
		{
			name: "unknown command",
			frame: protocol.FrameDTO{
				Cmd: ';',
			},
			wantValue:  []byte("unknown command"),
			wantStatus: protocol.STATUS_ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &testApp{}

			got := commandRouting(app, tt.frame)

			if got.Status != tt.wantStatus {
				t.Errorf("status: want %v, got %v", tt.wantStatus, got.Status)
			}

			if !bytes.Equal(tt.wantValue, got.Value) {
				t.Errorf("value: want %v, got %v", string(tt.wantValue), string(got.Value))
			}
		})
	}
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
				_ = app.set(tt.key, tt.initial, false, false)
			}
			got := app.set(tt.key, tt.value, tt.overwrite, tt.old)

			if string(got.Value) != string(tt.wantReturn) {
				t.Errorf("set() = %v, want %v", string(got.Value), string(tt.wantReturn))
			}
			gotMap := app.get(tt.key)
			if string(gotMap.Value) != string(tt.wantMap) {
				t.Errorf("get() = %v, want %v", string(gotMap.Value), string(tt.wantMap))
			}
		})
	}
}

func TestGet(t *testing.T) {
	app := newTestApp()

	want := []byte("Jack")

	_ = app.set("cat", want, false, false)

	got := app.get("cat")

	if !bytes.Equal(got.Value, want) {
		t.Errorf("expected %v, got %v", want, got.Value)
	}
}

func TestDel(t *testing.T) {
	app := newTestApp()

	want := []byte("Jack")

	_ = app.set("cat", want, false, false)

	got := app.del("cat")

	if !bytes.Equal(want, got.Value) {
		t.Errorf("delete return - want %v, got %v", want, got.Value)
	}

	gotAfterDel := app.get("cat")
	if gotAfterDel.Status != protocol.STATUS_NOT_FOUND {
		t.Errorf("get after delete - want STATUS_NOT_FOUND, got status %v", gotAfterDel.Status)
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value []byte
		want  []byte
	}{
		{
			name:  "item exists",
			key:   "cat",
			value: []byte("jack"),
			want:  []byte("1"),
		},
		{
			name: "item doesn't exist",
			key:  "dog",
			want: []byte("0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp()

			if tt.value != nil {
				_ = app.set(tt.key, tt.value, false, false)
			}

			got := app.exists(tt.key)

			if !bytes.Equal(tt.want, got.Value) {
				t.Errorf("want %v, got %v", tt.want, got.Value)
			}
		})
	}
}
