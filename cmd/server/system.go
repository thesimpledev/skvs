package main

import (
	"encoding/json"
	"fmt"
)

type command int

const (
	SET command = iota
	GET
	DELETE
	EXISTS
)

func (c command) String() string {
	switch c {
	case SET:
		return "SET"
	case GET:
		return "GET"
	case DELETE:
		return "DELETE"
	case EXISTS:
		return "EXISTS"
	default:
		return "UNKNOWN"
	}
}

func (c *command) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "SET":
			*c = SET
			return nil
		case "GET":
			*c = GET
			return nil
		case "DELETE":
			*c = DELETE
			return nil
		case "EXISTS":
			*c = EXISTS
			return nil
		default:
			return fmt.Errorf("unknown command %q", s)
		}
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*c = command(i)
		return nil
	}

	return fmt.Errorf("invalid command: %s", string(data))
}

type Message struct {
	Command   command `json:"command"`
	Key       string  `json:"key"`
	Value     string  `json:"value,omitempty"`
	Overwrite bool    `json:"overwrite,omitempty"`
	Old       bool    `json:"old,omitempty"`
}

func processMessage(message string) (string, error) {
	var input Message
	err := json.Unmarshal([]byte(message), &input)
	if err != nil {
		return "", err
	}

	switch input.Command {
	case SET:
		return set(input.Key, input.Value, input.Overwrite, input.Old)
	case GET:
		return get(input.Key)
	case DELETE:
		return del(input.Key)
	case EXISTS:
		return exists(input.Key)
	default:
		return "", fmt.Errorf("command %s", input.Command)
	}
}
