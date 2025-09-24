package main

import (
	"fmt"
	"net"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

func (app *application) serve() error {
	addr := fmt.Sprintf(":%d", app.cfg.port)

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP addr %s: %w", addr, err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer server.Close()

	app.log.Info("listening", "address", addr)

	buffer := make([]byte, protocol.EncryptedFrameSize)

	for {
		n, clientAddr, err := server.ReadFromUDP(buffer)
		if err != nil {
			app.log.Error("failed to read UDP packet", "err", err)
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])
		go app.handlePacket(server, clientAddr, data)
	}
}

func (app *application) handlePacket(server *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	payload, err := encryption.Decrypt(data)
	if err != nil {
		app.log.Error("Decrypt failed", "Err", err)
		app.sendMessage([]byte("ERROR: failed to process message"), server, clientAddr)
		return
	}

	response, err := processMessage(payload)
	if err != nil {
		app.log.Error("failed to process message", "err", err)
		app.sendMessage([]byte("ERROR: failed to process message"), server, clientAddr)
		return
	}

	encryptedResponse, err := encryption.Encrypt(response)
	if err != nil {
		app.log.Error("Encryption failed", "Err", err)
		app.sendMessage([]byte("ERROR: failed to process message"), server, clientAddr)
		return
	}

	app.sendMessage(encryptedResponse, server, clientAddr)
}

func (app *application) sendMessage(message []byte, server *net.UDPConn, clientAddr *net.UDPAddr) {
	_, err := server.WriteToUDP(message, clientAddr)
	if err != nil {
		app.log.Error("failed to write response", "err", err)
		return
	}
}
