package protocol

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestProtocol(t *testing.T) {
	tests := []struct {
		name      string
		cmd       string
		key       string
		value     string
		overwrite bool
		old       bool
		err       bool
	}{
		{
			name:      "successful new set frame",
			cmd:       "set",
			key:       "key",
			value:     "value",
			overwrite: false,
			old:       false,
		},
		{
			name:      "successful new set frame with overwrite",
			cmd:       "set",
			key:       "key",
			value:     "value",
			overwrite: true,
			old:       false,
		},
		{
			name:      "successful new set frame with old",
			cmd:       "set",
			key:       "key",
			value:     "value",
			overwrite: false,
			old:       true,
		},
		{
			name:      "successful new set frame with overwrite and old",
			cmd:       "set",
			key:       "key",
			value:     "value",
			overwrite: true,
			old:       true,
		},
		{
			name:      "failed new set key too long",
			cmd:       "set",
			key:       strings.Repeat("a", KeySize+1),
			value:     "value",
			overwrite: false,
			old:       false,
			err:       true,
		},
		{
			name:      "failed new set value too long",
			cmd:       "set",
			key:       "key",
			value:     strings.Repeat("a", ValueSize+1),
			overwrite: false,
			old:       false,
			err:       true,
		},
		{
			name: "successful new get",
			cmd:  "get",
			key:  "key",
		},
		{
			name: "successful new delete",
			cmd:  "delete",
			key:  "key",
		},
		{
			name: "successful new exists",
			cmd:  "exists",
			key:  "key",
		},
		{
			name:  "failed new set key empty",
			cmd:   "set",
			value: "value",
			err:   true,
		},
		{
			name: "failed new set value empty",
			cmd:  "set",
			key:  "key",
			err:  true,
		},
		{
			name: "failed new get key empty",
			cmd:  "get",
			err:  true,
		},
		{
			name: "failed new delete key empty",
			cmd:  "delete",
			err:  true,
		},
		{
			name: "failed new exists key empty",
			cmd:  "exists",
			err:  true,
		},
		{
			name: "failed invalid command",
			cmd:  "mycommand",
			err:  true,
		},
		{
			name:  "failed new set key with NUL byte",
			cmd:   "set",
			key:   "ke\x00y",
			value: "value",
			err:   true,
		},
		{
			name:  "failed new set value with NUL byte",
			cmd:   "set",
			key:   "key",
			value: "val\x00ue",
			err:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto, err := NewFrameDTO(tt.cmd, tt.key, tt.value, tt.overwrite, tt.old)
			if (err != nil) != tt.err {
				t.Fatalf("NewFrameDTO() error = %v, wantErr %v", err, tt.err)
			}

			if tt.err {
				return
			}

			frame := DtoToFrame(dto)

			got, err := FrameToDTO(frame)
			if err != nil {
				t.Fatalf("failed to created dto from frame: %v", err)
			}

			if !reflect.DeepEqual(dto, got) {
				t.Errorf("got %+v, want %+v", got, dto)
			}
		})
	}
}

func TestFrameToLarge(t *testing.T) {
	frame := make([]byte, FrameSize+1)
	_, err := FrameToDTO(frame)

	if err == nil {
		t.Errorf("frame should return size error")
	}
}

func TestResponseDTORoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		status byte
		value  []byte
	}{
		{name: "ok with value", status: STATUS_OK, value: []byte("value")},
		{name: "not found with nil value", status: STATUS_NOT_FOUND, value: nil},
		{name: "error with message", status: STATUS_ERROR, value: []byte("unknown command")},
		{name: "ok with full value", status: STATUS_OK, value: bytes.Repeat([]byte("a"), ResponseValueSize)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := ResponseDTOToFrame(NewResponseDTO(tt.status, tt.value))
			if len(frame) != FrameSize {
				t.Fatalf("response frame size = %d, want %d", len(frame), FrameSize)
			}

			got, err := FrameToResponseDTO(frame)
			if err != nil {
				t.Fatalf("failed to create response dto from frame: %v", err)
			}

			if got.Status != tt.status {
				t.Errorf("status = %d, want %d", got.Status, tt.status)
			}

			if !bytes.Equal(got.Value, tt.value) {
				t.Errorf("value = %q, want %q", got.Value, tt.value)
			}
		})
	}
}

func TestFrameToResponseDTOWrongSize(t *testing.T) {
	_, err := FrameToResponseDTO(make([]byte, FrameSize+1))

	if err == nil {
		t.Errorf("response frame should return size error")
	}
}
