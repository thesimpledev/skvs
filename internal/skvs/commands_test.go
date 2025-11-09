package skvs

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type testApp struct{}

func (app *testApp) set(_ string, _ []byte, _, _ bool) []byte {
	return []byte("set")
}

func (app *testApp) get(_ string) []byte {
	return []byte("get")
}

func (app *testApp) del(_ string) []byte {
	return []byte("del")
}

func (app *testApp) exists(_ string) []byte {
	return []byte("exists")
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
		name  string
		frame protocol.FrameDTO
		want  []byte
		err   bool
	}{
		{
			name: "set command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_SET,
			},
			want: []byte("set"),
			err:  false,
		},
		{
			name: "get command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_GET,
			},
			want: []byte("get"),
			err:  false,
		},
		{
			name: "del command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_DELETE,
			},
			want: []byte("del"),
			err:  false,
		},
		{
			name: "exists command",
			frame: protocol.FrameDTO{
				Cmd: protocol.CMD_EXISTS,
			},
			want: []byte("exists"),
			err:  false,
		},
		{
			name: "unknown command",
			frame: protocol.FrameDTO{
				Cmd: ';',
			},
			want: []byte(""),
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &testApp{}

			got, err := commandRouting(app, tt.frame)
			if err != nil && !tt.err {
				t.Errorf("wanted no error, got %v", err.Error())
			}

			if !bytes.Equal(tt.want, got) {
				t.Errorf("want: %v, got: %v", string(tt.want), string(got))
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

			if string(got) != string(tt.wantReturn) {
				t.Errorf("set() = %v, want %v", string(got), string(tt.wantReturn))
			}
			gotMap := app.get(tt.key)
			if string(gotMap) != string(tt.wantMap) {
				t.Errorf("get() = %v, want %v", string(gotMap), string(tt.wantMap))
			}
		})
	}
}

func TestGet(t *testing.T) {
	app := newTestApp()

	want := []byte("Jack")

	_ = app.set("cat", want, false, false)

	got := app.get("cat")

	if !bytes.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDel(t *testing.T) {
	app := newTestApp()

	want := []byte("Jack")

	_ = app.set("cat", want, false, false)

	got := app.del("cat")

	if !bytes.Equal(want, got) {
		t.Errorf("delete return - want %v, got %v", want, got)
	}

	gotAfterDel := app.get("cat")
	if !bytes.Equal(gotAfterDel, nil) {
		t.Errorf("get after delete - want nil, got %v", gotAfterDel)
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

			if !bytes.Equal(tt.want, got) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
