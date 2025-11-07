package main

import (
	"net"
)

func (app *application) handlePacket(server *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	payload, err := app.encryptor.Decrypt(data)
	if err != nil {
		app.log.Error("Decrypt failed", "Err", err)
		app.sendMessage([]byte("ERROR: failed to process message"), server, clientAddr)
		return
	}

	response, err := app.processMessage(payload)
	if err != nil {
		app.log.Error("failed to process message", "err", err)
		app.sendMessage([]byte("ERROR: failed to process message"), server, clientAddr)
		return
	}

	encryptedResponse, err := app.encryptor.Encrypt(response)
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
