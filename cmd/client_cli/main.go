package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

func main() {
	overwrite := flag.Bool("overwrite", false, "Allow overwriting existing values")
	old := flag.Bool("old", false, "Return the previous value if available")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: cli <set|get|delete|exists> <key> [value] [--overwrite] [--old]")
		os.Exit(1)
	}

	commandStr := args[0]
	key := args[1]
	value := ""
	if len(args) > 2 {
		value = args[2]
	}

	c, err := client.New(fmt.Sprintf("localhost:%d", protocol.Port))
	if err != nil {
		fmt.Println("Error creating client:", err)
		os.Exit(1)
	}
	defer c.Close()

	var cmd byte
	switch commandStr {
	case "set":
		cmd = protocol.CMD_SET
	case "get":
		cmd = protocol.CMD_GET
	case "delete":
		cmd = protocol.CMD_DELETE
	case "exists":
		cmd = protocol.CMD_EXISTS
	default:
		fmt.Println("Unknown command:", commandStr)
		os.Exit(1)
	}

	var flags uint32
	if *overwrite {
		flags |= protocol.FLAG_OVERWRITE
	}
	if *old {
		flags |= protocol.FLAG_OLD
	}

	ctx, cancel := context.WithTimeout(context.Background(), protocol.Timeout)
	defer cancel()

	resp, err := c.Send(ctx, cmd, flags, key, value)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Response:", resp)
}
