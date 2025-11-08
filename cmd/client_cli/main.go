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

	dto, err := protocol.NewFrameDTO(commandStr, key, value, *overwrite, *old)
	if err != nil {
		fmt.Printf("error creating data transfer object: %v\n", err)
	}

	c, err := client.New(fmt.Sprintf("localhost:%d", protocol.Port), nil)
	if err != nil {
		fmt.Println("Error creating client:", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), protocol.Timeout)
	defer cancel()

	resp, err := c.Send(ctx, dto)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Response:", resp)
}
