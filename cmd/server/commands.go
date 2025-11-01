package main

import "bytes"

func (app *application) set(key string, value []byte, overwrite, old bool) ([]byte, error) {
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
	return bytes.Clone(returnValue), nil
}

func (app *application) get(key string) ([]byte, error) {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return bytes.Clone(app.skvs[key]), nil
}

func (app *application) del(key string) ([]byte, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	returnValue := app.skvs[key]
	delete(app.skvs, key)
	return bytes.Clone(returnValue), nil
}

func (app *application) exists(key string) ([]byte, error) {
	app.mu.RLock()
	defer app.mu.RUnlock()
	if _, exists := app.skvs[key]; exists {
		return []byte("1"), nil
	}

	return []byte("0"), nil
}
