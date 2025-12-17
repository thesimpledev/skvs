// Package protocol provides the protocol definitions for the key-value store.
package protocol

import (
	"bytes"
	"fmt"
	"time"
)

const (
	CMD_SET    = 0
	CMD_GET    = 1
	CMD_DELETE = 2
	CMD_EXISTS = 3

	STATUS_OK        = 0
	STATUS_NOT_FOUND = 1
	STATUS_ERROR     = 2

	FLAG_OVERWRITE uint32 = 1 << 0
	FLAG_OLD       uint32 = 1 << 1

	CommandSize        = 1
	FlagSize           = 4
	StatusSize         = 1
	FrameSize          = 996
	EncryptedFrameSize = 1024
	KeySize            = 128
	ValueSize          = 863
	ResponseValueSize  = FrameSize - StatusSize
	Port               = 4040
	Timeout            = 5 * time.Second
)

type FrameDTO struct {
	Cmd       byte
	Key       string
	Value     []byte
	Overwrite bool
	Old       bool
}

func NewFrameDTO(cmdStr string, key, value string, overwrite, old bool) (FrameDTO, error) {
	var cmd byte
	switch cmdStr {
	case "set":
		cmd = CMD_SET
	case "get":
		cmd = CMD_GET
	case "delete":
		cmd = CMD_DELETE
	case "exists":
		cmd = CMD_EXISTS
	default:
		return FrameDTO{}, fmt.Errorf("unknown command string %s", cmdStr)
	}

	if key == "" {
		return FrameDTO{}, fmt.Errorf("key cannot be empty")
	}

	if cmd == CMD_SET && value == "" {
		return FrameDTO{}, fmt.Errorf("value cannot be empty in set command")
	}

	if len(key) > KeySize {
		return FrameDTO{}, fmt.Errorf("key too long. size is %d and max size allowed is %d", len(key), KeySize)
	}

	if len(value) > ValueSize {
		return FrameDTO{}, fmt.Errorf("value too long. size is %d and max size allowed is %d", len(value), ValueSize)
	}

	byteValue := []byte(value)

	dto := FrameDTO{
		Cmd:       cmd,
		Key:       key,
		Value:     byteValue,
		Overwrite: overwrite,
		Old:       old,
	}

	return dto, nil
}

func FrameToDTO(frame []byte) (FrameDTO, error) {
	if len(frame) != FrameSize {
		return FrameDTO{}, fmt.Errorf("invalid frame size %d", len(frame))
	}

	cmd := frame[0]

	flags := uint32(frame[1]) |
		uint32(frame[2])<<8 |
		uint32(frame[3])<<16 |
		uint32(frame[4])<<24

	start := CommandSize + FlagSize
	keyBytes := frame[start : start+KeySize]
	valBytes := frame[start+KeySize : start+KeySize+ValueSize]

	overwrite := flags&FLAG_OVERWRITE != 0
	old := flags&FLAG_OLD != 0

	key := string(bytes.TrimRight(keyBytes, "\x00"))
	value := bytes.TrimRight(valBytes, "\x00")

	frameDTO := FrameDTO{
		Cmd:       cmd,
		Key:       key,
		Value:     value,
		Overwrite: overwrite,
		Old:       old,
	}

	return frameDTO, nil
}

func DtoToFrame(dto FrameDTO) []byte {
	frame := make([]byte, FrameSize)
	var flags uint32

	if dto.Overwrite {
		flags |= FLAG_OVERWRITE
	}

	if dto.Old {
		flags |= FLAG_OLD
	}

	// Building the frame manually. While I could use the encoding/binary package I decided doing it by hand would be more clear.
	frame[0] = dto.Cmd
	frame[1] = byte(flags)
	frame[2] = byte(flags >> 8)
	frame[3] = byte(flags >> 16)
	frame[4] = byte(flags >> 24)

	copy(frame[5:5+KeySize], []byte(dto.Key))
	copy(frame[5+KeySize:], []byte(dto.Value))

	return frame
}

type ResponseDTO struct {
	Status byte
	Value  []byte
}

func NewResponseDTO(status byte, value []byte) ResponseDTO {
	return ResponseDTO{
		Status: status,
		Value:  value,
	}
}

func ResponseDTOToFrame(dto ResponseDTO) []byte {
	frame := make([]byte, FrameSize)
	frame[0] = dto.Status
	copy(frame[StatusSize:], dto.Value)
	return frame
}

func FrameToResponseDTO(frame []byte) (ResponseDTO, error) {
	if len(frame) != FrameSize {
		return ResponseDTO{}, fmt.Errorf("invalid response frame size %d", len(frame))
	}

	status := frame[0]
	value := bytes.TrimRight(frame[StatusSize:], "\x00")

	return ResponseDTO{
		Status: status,
		Value:  value,
	}, nil
}
