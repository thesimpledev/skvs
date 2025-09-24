package main

import (
	"log/slog"
	"os"
	"strconv"
)

type config struct {
	port int
}

type application struct {
	cfg *config
	log *slog.Logger
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		logger.Info("invalid or missing PORT, defaulting to 4000")
		port = 4000
	}

	config := &config{
		port: port,
	}

	app := &application{
		cfg: config,
		log: logger,
	}

	app.log.Info("Launching", "port", port)

	err = app.serve()
	if err != nil {
		app.log.Error("server error", "err", err)
		os.Exit(1)
	}

}
