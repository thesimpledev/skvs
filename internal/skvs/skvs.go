// Package skvs is the core domain logic for Simple Key Value Store
package skvs

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type SKVS interface {
	set(key string, value []byte, overwrite, old bool) []byte
	get(key string) []byte
	del(key string) []byte
	exists(key string) []byte
}

type App struct {
	log  *slog.Logger
	skvs map[string][]byte
	mu   sync.RWMutex
}

func New(log *slog.Logger) *App {
	return &App{
		log:  log,
		skvs: make(map[string][]byte, 0),
	}
}

func ProcessMessage(app *App, frame []byte) ([]byte, error) {
	frameDTO, err := protocol.FrameToDTO(frame)
	if err != nil {
		return nil, fmt.Errorf("unable to parse frame: %v", err)
	}
	return commandRouting(app, frameDTO)
}
