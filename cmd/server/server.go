package main

import (
	"fmt"
	"net"
)

func (app *application) serve() error {
	addr := fmt.Sprintf(":%d", app.cfg.port)

	server, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer server.Close()

	app.log.Info("listening", "address", addr)

	for {
		connection, err := server.Accept()
		if err != nil {
			app.log.Error("failed to accept connection", "err", err)
			continue
		}

		go app.handleConnection(connection)
	}

}

func (app *application) handleConnection(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, 1024)
	bytesRead, err := connection.Read(buffer)
	if err != nil {
		app.log.Error("failed to read from connection", "err", err)
		return
	}

	encodedPayload := string(buffer[:bytesRead])

	decodedPayload := app.decrypt(encodedPayload)
	response, err := processMessage(decodedPayload)
	if err != nil {
		app.log.Error("failed to process message", "err", err)
		app.sendMessage("ERROR: failed to process message", connection)
		return
	}

	app.sendMessage(response, connection)

}

func (app *application) sendMessage(message string, connection net.Conn) {
	_, err := connection.Write([]byte(message))
	if err != nil {
		app.log.Error("failed to write response", "err", err)
		return
	}
}

func (app *application) decrypt(payload string) string {
	return payload
}
