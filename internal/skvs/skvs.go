// Package skvs is the core domain logic for Simple Key Value Store
package skvs

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type SKVS interface {
	set(key string, value []byte, overwrite, old bool) protocol.ResponseDTO
	get(key string) protocol.ResponseDTO
	del(key string) protocol.ResponseDTO
	exists(key string) protocol.ResponseDTO
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
	responseDTO := commandRouting(app, frameDTO)
	return protocol.ResponseDTOToFrame(responseDTO), nil
}
