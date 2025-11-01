package main

import "bytes"

func set(key string, value []byte, overwrite, old bool) ([]byte, error) {
	var returnValue []byte
	mu.Lock()
	defer mu.Unlock()
	if old {
		returnValue = skvs[key]
	}

	if oldValue, exists := skvs[key]; !exists || overwrite {
		if !old {
			returnValue = value
		}

		skvs[key] = bytes.Clone(value)
	} else {
		returnValue = oldValue
	}

	return bytes.Clone(returnValue), nil
}

func get(key string) ([]byte, error) {
	mu.RLock()
	defer mu.RUnlock()
	return bytes.Clone(skvs[key]), nil
}

func del(key string) ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	returnValue := skvs[key]
	delete(skvs, key)
	return bytes.Clone(returnValue), nil
}

func exists(key string) ([]byte, error) {
	mu.RLock()
	defer mu.RUnlock()
	if _, exists := skvs[key]; exists {
		return []byte("1"), nil
	}

	return []byte("0"), nil
}
