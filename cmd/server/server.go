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
			return
		default:
			bufPtr := bufPool.Get().(*[]byte)
			buf := *bufPtr

			_ = s.conn.SetReadDeadline(time.Now().Add(s.readTimeout))

			n, clientAddr, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				bufPool.Put(bufPtr)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				s.log.Error("failed to read UDP packet", "err", err)
				continue
			}

			data := make([]byte, n)
			copy(data, buf[:n])
			bufPool.Put(bufPtr)
			select {
			case s.semaphore <- struct{}{}:

				go func() {
					defer func() { <-s.semaphore }()
					s.handlePacket(clientAddr, data)
				}()
			default:
				s.log.Warn("request dropped - at capacity", "addr", clientAddr)
			}

		}
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
		s.sendMessage([]byte("ERROR: failed to process message"), s.conn, clientAddr)
		return
	}

	response, err := skvs.ProcessMessage(s.app, payload)
	if err != nil {
		s.log.Error("failed to process message", "err", err)
		s.sendMessage([]byte("ERROR: failed to process message"), s.conn, clientAddr)
		return
	}

	encryptedResponse, err := s.encryptor.Encrypt(response)
	if err != nil {
		s.log.Error("Encryption failed", "Err", err)
		s.sendMessage([]byte("ERROR: failed to process message"), s.conn, clientAddr)
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
