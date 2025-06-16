package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

func (s *ChatServer) handleJoinLeaveChatroom(conn net.Conn, chatroom string, operasi string) {
	s.mu.Lock()
	clientName := s.clients[conn]
	s.mu.Unlock()

	switch operasi {
	case "join":
		s.mu.Lock()
		s.rooms[chatroom][conn] = false // false berarti user join tapi tidak aktif di dalamnya
		s.mu.Unlock()
		conn.Write([]byte(fmt.Sprintf("Anda telah bergabung dalam chatroom '%s'.\n", chatroom)))
		s.broadcastPerRoom(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah bergabung dalam chatroom '%s'.", clientName, chatroom), chatroom, conn)
	case "leave":
		s.mu.Lock()
		delete(s.rooms[chatroom], conn)
		s.mu.Unlock()
		conn.Write([]byte(fmt.Sprintf("Anda telah meninggalkan chatroom '%s'.\n", chatroom)))
		s.broadcastPerRoom(fmt.Sprintf("\n[PEMBERITAHUAN] %s tidak lagi tergabung dalam chatroom '%s'.", clientName, chatroom), chatroom, conn)
	}
}

func (s *ChatServer) handleChatroom(conn net.Conn, wg *sync.WaitGroup, roomName string) {
	defer wg.Done()
	s.mu.Lock()
	clientName := s.clients[conn]
	s.mu.Unlock()

	// Set status client menjadi aktif di room ini
	s.mu.Lock()
	s.rooms[roomName][conn] = true
	s.mu.Unlock()

	reader := bufio.NewReader(conn)
	conn.Write([]byte(fmt.Sprintf("\n--- Selamat datang di room [%s] ---\nKetik pesan Anda. Ketik '/kembali' untuk keluar dari room dan kembali ke menu utama.\n", roomName)))
	s.broadcastPerRoom(fmt.Sprintf("[%s] telah masuk ke dalam '%s'.", clientName, roomName), roomName, conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			s.handleDisconnect(conn)
			return
		}
		message = strings.TrimSpace(message)

		if message == "/kembali" {
			break // Keluar dari loop untuk kembali ke menu utama
		}

		if message != "" {
			s.broadcastPerRoom(fmt.Sprintf("[%s]: %s", clientName, message), roomName, conn)
		}
	}

	// Set status client menjadi tidak aktif lagi di room ini
	s.mu.Lock()
	// Pastikan koneksi masih ada sebelum mengubah status
	if _, ok := s.rooms[roomName]; ok {
		s.rooms[roomName][conn] = false
	}
	s.mu.Unlock()

	conn.Write([]byte(fmt.Sprintf("Anda telah keluar dari percakapan chatroom '%s'.\n", roomName)))
	s.broadcastPerRoom(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah keluar dari percakapan.", clientName), roomName, conn)
}

func (s *ChatServer) broadcastPerRoom(message string, roomName string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Pastikan room masih ada
	if room, ok := s.rooms[roomName]; ok {
		for conn, isActive := range room {
			// Kirim ke semua orang di room kecuali pengirim
			if conn != sender && isActive {
				_, err := conn.Write([]byte(message + "\n"))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to send message to %s: %v\n", s.clients[conn], err)
				}
			}
		}
	}
}

func (s *ChatServer) handleCreateChatroom(conn net.Conn, chatroomName string) {
	s.mu.Lock()
	s.rooms[chatroomName] = make(map[net.Conn]bool)
	// Otomatis join setelah membuat, tapi tidak aktif di dalamnya
	s.rooms[chatroomName][conn] = false
	clientName := s.clients[conn]
	s.mu.Unlock()

	conn.Write([]byte(fmt.Sprintf("Chatroom '%s' telah berhasil dibuat dan Anda otomatis bergabung.\n", chatroomName)))
	s.broadcast(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah membuat chatroom baru: %s\n", clientName, chatroomName), conn)
}
