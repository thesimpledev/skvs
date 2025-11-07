package main

import (
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
	"github.com/thesimpledev/skvs/internal/skvs"
)

type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type server struct {
	log       *slog.Logger
	encryptor Encryptor
	conn      *net.UDPConn
	port      string
	app       *skvs.App
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	e, err := encryption.New(nil)
	if err != nil {
		logger.Error("Unable to create Encryptor: ", "err", err)
		os.Exit(1)
	}

	server := &server{
		log:       logger,
		encryptor: e,
	}
	server.port = os.Getenv("PORT")
	server.app = skvs.New(logger)

	udpConn, err := server.startUDPServer()
	if err != nil {
		logger.Error("unable to start UDP Server", "err", err)
		os.Exit(1)
	}
	server.conn = udpConn
	defer func() {
		_ = udpConn.Close()
	}()

	server.serverListen()
}

func (s *server) serverListen() {
	bufPool := sync.Pool{
		New: func() any {
			buf := make([]byte, protocol.EncryptedFrameSize)
			return &buf
		},
	}

	for {
		bufPtr := bufPool.Get().(*[]byte)
		buf := *bufPtr
		n, clientAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			s.log.Error("failed to read UDP packet", "err", err)
			bufPool.Put(bufPtr)
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])
		bufPool.Put(bufPtr)
		go func() {
			s.handlePacket(clientAddr, data)
		}()
	}
}

func (s *server) startUDPServer() (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", ":"+s.port)
	if err != nil {
		return nil, err
	}

	server, err := net.ListenUDP("udp", udpAddr)
	return server, err
}
