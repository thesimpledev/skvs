package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/thesimpledev/skvs/internal/client"
	"github.com/thesimpledev/skvs/internal/protocol"
)

const usage = "Usage: cli [--overwrite] [--old] <set|get|delete|exists> <key> [value]"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dto, err := parseArgs()
	if err != nil {
		return err
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", protocol.Port)
	}

	c, err := client.New("localhost:"+port, []byte(os.Getenv("SKVS_ENCRYPTION_KEY")))
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), protocol.Timeout)
	defer cancel()

	resp, err := c.Send(ctx, dto)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	fmt.Println("Response:", resp)
	return nil
}

func parseArgs() (protocol.FrameDTO, error) {
	overwrite := flag.Bool("overwrite", false, "Allow overwriting existing values")
	old := flag.Bool("old", false, "Return the previous value if available")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		return protocol.FrameDTO{}, fmt.Errorf("%s", usage)
	}

	commandStr := args[0]
	key := args[1]
	value := ""
	if len(args) > 2 {
		value = args[2]
	}

	dto, err := protocol.NewFrameDTO(commandStr, key, value, *overwrite, *old)
	if err != nil {
		return protocol.FrameDTO{}, fmt.Errorf("%w\n%s", err, usage)
	}

	return dto, nil
}
