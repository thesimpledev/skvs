package skvs

import "bytes"

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
