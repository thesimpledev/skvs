package skvs

import (
	"io"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := New(logger)

	if app == nil {
		t.Fatal("App is nill and should not be")
	}

	if app.skvs == nil {
		t.Error("skvs is nil and should not be ")
	}
}
