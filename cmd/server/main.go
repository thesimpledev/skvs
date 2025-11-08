//go:build exclude_tests

package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/skvs"
)

type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type server struct {
	log         *slog.Logger
	encryptor   Encryptor
	conn        *net.UDPConn
	port        string
	app         *skvs.App
	semaphore   chan struct{}
	readTimeout time.Duration
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	e, err := encryption.New([]byte(os.Getenv("SKVS_ENCRYPTION_KEY")))
	if err != nil {
		logger.Error("Unable to create Encryptor: ", "err", err)
		os.Exit(1)
	}

	server := &server{
		log:         logger,
		encryptor:   e,
		readTimeout: 100 * time.Millisecond,
	}
	server.port = os.Getenv("PORT")
	server.app = skvs.New(logger)
	server.semaphore = make(chan struct{}, 1000)

	udpConn, err := server.startUDPServer()
	if err != nil {
		logger.Error("unable to start UDP Server", "err", err)
		os.Exit(1)
	}
	server.conn = udpConn
	defer func() {
		_ = udpConn.Close()
	}()

	server.serverListen(ctx)
}
