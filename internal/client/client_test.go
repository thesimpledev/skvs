package client

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thesimpledev/skvs/internal/encryption"
	"github.com/thesimpledev/skvs/internal/protocol"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

type stubHandler func(attempt int64, req protocol.FrameDTO) (protocol.ResponseDTO, bool)

type stubServer struct {
	conn      *net.UDPConn
	encryptor *encryption.Encryptor
	handler   stubHandler
	attempts  atomic.Int64
	done      chan struct{}
}

func startStub(t *testing.T, handler stubHandler) *stubServer {
	t.Helper()

	e, err := encryption.New(testKey)
	if err != nil {
		t.Fatalf("encryption.New() error = %v", err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP() error = %v", err)
	}

	s := &stubServer{conn: conn, encryptor: e, handler: handler, done: make(chan struct{})}
	go s.serve()

	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
		<-s.done
	})

	return s
}

func (s *stubServer) addr() string {
	return s.conn.LocalAddr().String()
}

func (s *stubServer) serve() {
	defer close(s.done)

	buf := make([]byte, protocol.EncryptedFrameSize)
	for {
		n, from, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		s.respond(buf[:n], from)
	}
}

func (s *stubServer) respond(data []byte, from *net.UDPAddr) {
	attempt := s.attempts.Add(1)

	payload, err := s.encryptor.Decrypt(data)
	if err != nil {
		return
	}

	req, err := protocol.FrameToDTO(payload)
	if err != nil {
		return
	}

	resp, reply := s.handler(attempt, req)
	if !reply {
		return
	}

	encrypted, err := s.encryptor.Encrypt(protocol.ResponseDTOToFrame(resp))
	if err != nil {
		return
	}

	_, err = s.conn.WriteToUDP(encrypted, from)
	if err != nil {
		return
	}
}

func newTestClient(t *testing.T, addr string) *Client {
	t.Helper()

	c, err := New(addr, testKey)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func getFrame(t *testing.T) protocol.FrameDTO {
	t.Helper()

	dto, err := protocol.NewFrameDTO("get", "key", "", false, false)
	if err != nil {
		t.Fatalf("NewFrameDTO() error = %v", err)
	}
	return dto
}

func alwaysReply(resp protocol.ResponseDTO) stubHandler {
	return func(_ int64, _ protocol.FrameDTO) (protocol.ResponseDTO, bool) {
		return resp, true
	}
}

func TestSendResponses(t *testing.T) {
	tests := []struct {
		name     string
		response protocol.ResponseDTO
		want     string
		wantErr  string
	}{
		{
			name:     "ok",
			response: protocol.NewResponseDTO(protocol.STATUS_OK, []byte("value")),
			want:     "value",
		},
		{
			name:     "not found is empty with no error",
			response: protocol.NewResponseDTO(protocol.STATUS_NOT_FOUND, nil),
			want:     "",
		},
		{
			name:     "server error",
			response: protocol.NewResponseDTO(protocol.STATUS_ERROR, []byte("unknown command")),
			wantErr:  "unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := startStub(t, alwaysReply(tt.response))
			c := newTestClient(t, stub.addr())

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			got, err := c.Send(ctx, getFrame(t))
			if (err != nil) != (tt.wantErr != "") {
				t.Fatalf("Send() error = %v, wantErr %q", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Send() error = %v, want error containing %q", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Send() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSendDeliversFrame(t *testing.T) {
	stub := startStub(t, func(_ int64, req protocol.FrameDTO) (protocol.ResponseDTO, bool) {
		if req.Cmd != protocol.CMD_SET || req.Key != "key" || !req.Overwrite || req.Old {
			return protocol.NewResponseDTO(protocol.STATUS_ERROR, []byte("unexpected frame")), true
		}
		return protocol.NewResponseDTO(protocol.STATUS_OK, req.Value), true
	})
	c := newTestClient(t, stub.addr())

	dto, err := protocol.NewFrameDTO("set", "key", "value", true, false)
	if err != nil {
		t.Fatalf("NewFrameDTO() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.Send(ctx, dto)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got != "value" {
		t.Errorf("Send() = %q, want %q", got, "value")
	}
}

func TestSendRequiresDeadline(t *testing.T) {
	stub := startStub(t, alwaysReply(protocol.NewResponseDTO(protocol.STATUS_OK, []byte("value"))))
	c := newTestClient(t, stub.addr())

	_, err := c.Send(context.Background(), getFrame(t))
	if err == nil {
		t.Fatal("Send() without a deadline should fail")
	}
	if got := stub.attempts.Load(); got != 0 {
		t.Errorf("stub received %d datagrams, want 0", got)
	}
}

func TestSendTimesOutWithoutResponse(t *testing.T) {
	stub := startStub(t, func(_ int64, _ protocol.FrameDTO) (protocol.ResponseDTO, bool) {
		return protocol.ResponseDTO{}, false
	})
	c := newTestClient(t, stub.addr())

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_, err := c.Send(ctx, getFrame(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestSendRetriesAfterLostDatagram(t *testing.T) {
	stub := startStub(t, func(attempt int64, _ protocol.FrameDTO) (protocol.ResponseDTO, bool) {
		if attempt == 1 {
			return protocol.ResponseDTO{}, false
		}
		return protocol.NewResponseDTO(protocol.STATUS_OK, []byte("value")), true
	})
	c := newTestClient(t, stub.addr())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.Send(ctx, getFrame(t))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got != "value" {
		t.Errorf("Send() = %q, want %q", got, "value")
	}
	if attempts := stub.attempts.Load(); attempts != 2 {
		t.Errorf("stub received %d datagrams, want 2", attempts)
	}
}
