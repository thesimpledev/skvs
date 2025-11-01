package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type config struct {
	port int
}

type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type application struct {
	cfg       *config
	log       *slog.Logger
	encryptor Encryptor
	skvs      map[string][]byte
	mu        sync.RWMutex
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		logger.Info(fmt.Sprintf("invalid or missing PORT, defaulting to %d", protocol.Port))
		port = protocol.Port
	}

	config := &config{
		port: port,
	}

	e, err := encryption.New(nil)
	if err != nil {
		logger.Error("Unable to create Encryptor: ", "err", err)
		os.Exit(1)
	}

	app := &application{
		cfg:       config,
		log:       logger,
		encryptor: e,
		skvs:      make(map[string][]byte),
	}

	app.log.Info("Launching", "port", port)

	err = app.serve()
	if err != nil {
		app.log.Error("server error", "err", err)
		os.Exit(1)
	}
}
