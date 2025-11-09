package skvs

import (
	"io"
	"log/slog"
	"testing"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := New(logger)

	if app == nil {
		t.Fatal("App is nill and should not be")
	}
}

func TestProcessMessage(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
		err   bool
	}{
		{
			name:  "valid frame",
			frame: protocol.DtoToFrame(protocol.FrameDTO{Cmd: protocol.CMD_SET, Key: "key", Value: []byte("value")}),
			err:   false,
		},
		{
			name:  "invalid frame size",
			frame: make([]byte, 100),
			err:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp()

			_, err := ProcessMessage(app, tt.frame)
			if (err != nil) != tt.err {
				t.Errorf("ProcessMessage() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}
