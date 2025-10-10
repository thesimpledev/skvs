// Package protocol provides the protocol definitions for the key-value store.
package protocol

import "time"

const (
	CMD_SET    = 0
	CMD_GET    = 1
	CMD_DELETE = 2
	CMD_EXISTS = 3

	FLAG_OVERWRITE = 1 << 0
	FLAG_OLD       = 1 << 1

	CommandSize        = 1
	FlagSize           = 4
	FrameSize          = 996
	EncryptedFrameSize = 1024
	KeySize            = 128
	ValueSize          = 863
	Port               = 4040
	Timeout            = 5 * time.Second
)
