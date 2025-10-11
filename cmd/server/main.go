package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type config struct {
	port int
}

type application struct {
	cfg       *config
	log       *slog.Logger
	encryptor *encryption.Encryptor
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
	}

	app.log.Info("Launching", "port", port)

	err = app.serve()
	if err != nil {
		app.log.Error("server error", "err", err)
		os.Exit(1)
	}

}
