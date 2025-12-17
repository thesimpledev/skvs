package skvs

import (
	"bytes"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func commandRouting(app SKVS, frame protocol.FrameDTO) protocol.ResponseDTO {
	switch frame.Cmd {
	case protocol.CMD_SET:
		return app.set(frame.Key, frame.Value, frame.Overwrite, frame.Old)
	case protocol.CMD_GET:
		return app.get(frame.Key)
	case protocol.CMD_DELETE:
		return app.del(frame.Key)
	case protocol.CMD_EXISTS:
		return app.exists(frame.Key)
	default:
		return protocol.NewResponseDTO(protocol.STATUS_ERROR, []byte("unknown command"))
	}
}

func (app *App) set(key string, value []byte, overwrite, old bool) protocol.ResponseDTO {
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
	return protocol.NewResponseDTO(protocol.STATUS_OK, bytes.Clone(returnValue))
}

func (app *App) get(key string) protocol.ResponseDTO {
	app.mu.RLock()
	defer app.mu.RUnlock()
	value, exists := app.skvs[key]
	if !exists {
		return protocol.NewResponseDTO(protocol.STATUS_NOT_FOUND, nil)
	}
	return protocol.NewResponseDTO(protocol.STATUS_OK, bytes.Clone(value))
}

func (app *App) del(key string) protocol.ResponseDTO {
	app.mu.Lock()
	defer app.mu.Unlock()
	value, exists := app.skvs[key]
	delete(app.skvs, key)
	if !exists {
		return protocol.NewResponseDTO(protocol.STATUS_NOT_FOUND, nil)
	}
	return protocol.NewResponseDTO(protocol.STATUS_OK, bytes.Clone(value))
}

func (app *App) exists(key string) protocol.ResponseDTO {
	app.mu.RLock()
	defer app.mu.RUnlock()
	if _, exists := app.skvs[key]; exists {
		return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("1"))
	}
	return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("0"))
}
