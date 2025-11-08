package skvs

import (
	"bytes"
	"fmt"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func commandRouting(app SKVS, frame protocol.FrameDTO) ([]byte, error) {
	switch frame.Cmd {
	case protocol.CMD_SET:
		return app.set(frame.Key, frame.Value, frame.Overwrite, frame.Old), nil
	case protocol.CMD_GET:
		return app.get(frame.Key), nil
	case protocol.CMD_DELETE:
		return app.del(frame.Key), nil
	case protocol.CMD_EXISTS:
		return app.exists(frame.Key), nil
	default:
		return nil, fmt.Errorf("unknown command: %d", frame.Cmd)
	}
}

func (app *App) set(key string, value []byte, overwrite, old bool) []byte {
	var returnValue []byte
	var exists bool
	app.mu.Lock()
	defer app.mu.Unlock()

	if returnValue, exists = app.skvs[key]; !exists || overwrite {
		if !old {
			returnValue = value
		}
		app.skvs[key] = bytes.Clone(value)
	}
	if returnValue == nil {
		returnValue = []byte("")
	}
	return bytes.Clone(returnValue)
}

func (app *App) get(key string) []byte {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return bytes.Clone(app.skvs[key])
}

func (app *App) del(key string) []byte {
	app.mu.Lock()
	defer app.mu.Unlock()
	returnValue := app.skvs[key]
	delete(app.skvs, key)
	return bytes.Clone(returnValue)
}

func (app *App) exists(key string) []byte {
	app.mu.RLock()
	defer app.mu.RUnlock()
	if _, exists := app.skvs[key]; exists {
		return []byte("1")
	}

	return []byte("0")
}
