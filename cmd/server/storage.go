package main

import "sync"

var (
	skvs = make(map[string]string)
	mu   sync.RWMutex
)
