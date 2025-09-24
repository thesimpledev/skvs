package main

import "sync"

var (
	skvs = make(map[string][]byte)
	mu   sync.RWMutex
)
