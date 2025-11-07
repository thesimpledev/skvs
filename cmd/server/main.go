package main

import (
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type application struct {
	log       *slog.Logger
	encryptor Encryptor
	skvs      map[string][]byte
	mu        sync.RWMutex
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	port := os.Getenv("PORT")

	e, err := encryption.New(nil)
	if err != nil {
		logger.Error("Unable to create Encryptor: ", "err", err)
		os.Exit(1)
	}

	app := &application{
		log:       logger,
		encryptor: e,
		skvs:      make(map[string][]byte),
	}

	server, err := startUDPServer(port)
	if err != nil {
		logger.Error("unable to start UDP Server", "err", err)
		os.Exit(1)
	}

	defer func() {
		_ = server.Close()
	}()

	serverListen(server, app)
}

func serverListen(server *net.UDPConn, app *application) {
	bufPool := sync.Pool{
		New: func() any {
			buf := make([]byte, protocol.EncryptedFrameSize)
			return &buf
		},
	}

	for {
		bufPtr := bufPool.Get().(*[]byte)
		buf := *bufPtr
		n, clientAddr, err := server.ReadFromUDP(buf)
		if err != nil {
			app.log.Error("failed to read UDP packet", "err", err)
			bufPool.Put(bufPtr)
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])
		bufPool.Put(bufPtr)
		go func() {
			app.handlePacket(server, clientAddr, data)
		}()
	}
}

func startUDPServer(port string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}

	server, err := net.ListenUDP("udp", udpAddr)
	return server, err
}
