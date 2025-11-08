// Package skvs is the core domain logic for Simple Key Value Store
package skvs

import (
	"log/slog"
	"sync"
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
