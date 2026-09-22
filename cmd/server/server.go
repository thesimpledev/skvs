package main

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/thesimpledev/skvs/internal/protocol"
	"github.com/thesimpledev/skvs/internal/skvs"
)

func (s *server) serverListen(ctx context.Context) {
	bufPool := sync.Pool{
		New: func() any {
			buf := make([]byte, protocol.EncryptedFrameSize)
			return &buf
		},
	}

	for {
		select {
		case <-ctx.Done():
			s.handlers.Wait()
			return
		default:
			data, clientAddr, ok := s.readPacket(&bufPool)
			if !ok {
				continue
			}
			s.dispatch(clientAddr, data)
		}
	}
}

func (s *server) readPacket(bufPool *sync.Pool) ([]byte, *net.UDPAddr, bool) {
	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	buf := *bufPtr

	err := s.conn.SetReadDeadline(time.Now().Add(s.readTimeout))
	if err != nil {
		s.log.Error("failed to set read deadline", "err", err)
		return nil, nil, false
	}

	n, clientAddr, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		if netErr, isNetErr := err.(net.Error); isNetErr && netErr.Timeout() {
			return nil, nil, false
		}
		s.log.Error("failed to read UDP packet", "err", err)
		return nil, nil, false
	}

	data := make([]byte, n)
	copy(data, buf[:n])
	return data, clientAddr, true
}

func (s *server) dispatch(clientAddr *net.UDPAddr, data []byte) {
	select {
	case s.semaphore <- struct{}{}:
		s.handlers.Add(1)
		go func() {
			defer s.handlers.Done()
			defer func() { <-s.semaphore }()
			s.handlePacket(clientAddr, data)
		}()
	default:
		s.log.Warn("request dropped - at capacity", "addr", clientAddr)
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

func (s *server) handlePacket(clientAddr *net.UDPAddr, data []byte) {
	payload, err := s.encryptor.Decrypt(data)
	if err != nil {
		s.log.Error("Decrypt failed", "Err", err)
		return
	}

	response, err := skvs.ProcessMessage(s.app, payload)
	if err != nil {
		s.log.Error("failed to process message", "err", err)
		response = protocol.ResponseDTOToFrame(protocol.NewResponseDTO(protocol.STATUS_ERROR, []byte("failed to process message")))
	}

	encryptedResponse, err := s.encryptor.Encrypt(response)
	if err != nil {
		s.log.Error("Encryption failed", "Err", err)
		return
	}

	s.sendMessage(encryptedResponse, s.conn, clientAddr)
}

func (s *server) sendMessage(message []byte, server *net.UDPConn, clientAddr *net.UDPAddr) {
	_, err := server.WriteToUDP(message, clientAddr)
	if err != nil {
		s.log.Error("failed to write response", "err", err)
		return
	}
}
