package protocol

import (
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
			name:      "failed new set key to long",
			cmd:       "set",
			key:       strings.Repeat("a", KeySize+1),
			value:     "value",
			overwrite: false,
			old:       false,
			err:       true,
		},
		{
			name:      "failed new set value to long",
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
			name: "failed new get key emoty",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto, err := NewFrameDTO(tt.cmd, tt.key, tt.value, tt.overwrite, tt.old)
			if err != nil && !tt.err {
				t.Fatalf("failed to create new dto: %v", err)
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
