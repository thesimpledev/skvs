package main

import (
	"bytes"
	"fmt"

	"github.com/thesimpledev/skvs/internal/protocol"
)

func (app *application) processMessage(frame []byte) ([]byte, error) {
	if len(frame) != protocol.FrameSize {
		return nil, fmt.Errorf("invalid frame size %d", len(frame))
	}

	cmd := frame[0]

	flags := uint32(frame[1]) |
		uint32(frame[2])<<8 |
		uint32(frame[3])<<16 |
		uint32(frame[4])<<24

	start := protocol.CommandSize + protocol.FlagSize
	keyBytes := frame[start : start+protocol.KeySize]
	valBytes := frame[start+protocol.KeySize : start+protocol.KeySize+protocol.ValueSize]

	overwrite := flags&protocol.FLAG_OVERWRITE != 0
	old := flags&protocol.FLAG_OLD != 0

	key := string(bytes.TrimRight(keyBytes, "\x00"))
	value := bytes.TrimRight(valBytes, "\x00")

	switch cmd {
	case protocol.CMD_SET:
		return app.set(key, value, overwrite, old)
	case protocol.CMD_GET:
		return app.get(key)
	case protocol.CMD_DELETE:
		return app.del(key)
	case protocol.CMD_EXISTS:
		return app.exists(key)
	default:
		return nil, fmt.Errorf("unknown command: %d", cmd)
	}
}
