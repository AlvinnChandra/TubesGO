package main

import (
	"fmt"
	"net"
	"os"
)

func (s *ChatServer) broadcast(message string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.clients {
		if conn != sender {
			_, err := conn.Write([]byte(message))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send message to %s: %v\n", s.clients[conn], err)
			}
		}
	}
}

func (s *ChatServer) isNameTaken(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existingName := range s.clients {
		if existingName == name {
			return true
		}
	}
	return false
}

// Fungsi untuk menangani diskoneksi client
func (s *ChatServer) handleDisconnect(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cek apakah client ada di map
	if name, ok := s.clients[conn]; ok {
		fmt.Printf("%s disconnected.\n", name)

		// Hapus dari semua room
		for roomName := range s.rooms {
			delete(s.rooms[roomName], conn)
		}

		// Hapus dari daftar client
		delete(s.clients, conn)

		// Broadcast ke client lain bahwa user telah pergi
		// Jalankan broadcast di goroutine baru agar tidak deadlock
		go s.broadcast(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah meninggalkan aplikasi MariChatting.\n", name), conn)
	}
	conn.Close()
}
