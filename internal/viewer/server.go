package viewer

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	SocketPath string
	Hub        *Hub
}

func (s *Server) Listen(ctx context.Context) error {
	if s.Hub == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.SocketPath), 0o755); err != nil {
		return err
	}
	_ = os.Remove(s.SocketPath)
	ln, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
		_ = os.Remove(s.SocketPath)
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()
	return nil
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	enc := json.NewEncoder(conn)
	snap := Message{
		Type:      "snapshot",
		Adapter:   s.Hub.Adapter(),
		Telemetry: s.Hub.SnapshotFromProvider(context.Background(), 200),
		At:        time.Now().UTC(),
	}
	if err := enc.Encode(snap); err != nil {
		return
	}

	updates := s.Hub.Subscribe()
	defer s.Hub.Unsubscribe(updates)

	writer := bufio.NewWriter(conn)
	for msg := range updates {
		if err := enc.Encode(msg); err != nil {
			return
		}
		_ = writer.Flush()
	}
}
