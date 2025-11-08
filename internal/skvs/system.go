package skvs

import (
	"bytes"
	"fmt"

	"github.com/thesimpledev/skvs/internal/protocol"
)

type frameDTO struct {
	cmd       byte
	key       string
	value     []byte
	overwrite bool
	old       bool
}

func frameToDTO(frame []byte) (*frameDTO, error) {
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

	frameDTO := &frameDTO{
		cmd:       cmd,
		key:       key,
		value:     value,
		overwrite: overwrite,
		old:       old,
	}

	return frameDTO, nil
}

func (app *App) ProcessMessage(frame []byte) ([]byte, error) {
	frameDTO, err := frameToDTO(frame)
	if err != nil {
		return nil, fmt.Errorf("unable to parse frame: %v", err)
	}
	return app.commandRouting(frameDTO)
}
