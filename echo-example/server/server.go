package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

//Mengambil referensi dari
//https://gobyexample.com/mutexes

// Struct untuk map client dan mutex
type ChatServer struct {
	//digunakan untuk menghindari race condition
	mu sync.Mutex

	//Map untuk nyimpen koneksi client beserta nama
	clients map[net.Conn]string
	rooms   map[string]map[net.Conn]bool
}

func main() {
	// Inisialisasi ChatServer
	server := &ChatServer{
		clients: make(map[net.Conn]string),
		rooms:   make(map[string]map[net.Conn]bool),
	}
	server.rooms["general"] = make(map[net.Conn]bool)

	//Listener TCP port 9090
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server listening on :9090...")

	for {
		conn, err := ln.Accept() //Menerima koneksi dari client
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			continue
		}

		//Menangani koneksi client agar tidak deadlock
		go server.handleClient(conn)
	}
}

// Method receiver untuk menangani client
func (s *ChatServer) handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	var name string
	for {
		inputName, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read inputName!\n")
			// Jangan keluar dari server, cukup putuskan koneksi client ini
			return
		}
		inputName = strings.TrimSpace(inputName)

		// Cek apakah nama sudah digunakan
		if s.isNameTaken(inputName) {
			conn.Write([]byte("Nama pengguna sudah digunakan, silakan pilih nama lain.\n"))
			continue
		} else if inputName == "" {
			conn.Write([]byte("Nama pengguna tidak boleh kosong!\n"))
			continue
		} else {
			conn.Write([]byte(fmt.Sprintf("Selamat datang di MariChatting, %s!\n", inputName)))
			name = inputName
			break
		}
	}

	// Simpan client setelah username valid
	s.mu.Lock()
	s.clients[conn] = name
	s.rooms["general"][conn] = false
	s.mu.Unlock()

	fmt.Printf("New client connected: %s\n", name)
	s.broadcast(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah masuk ke aplikasi MariChatting.\n", name), conn)

	// Loop utama untuk menampilkan menu setelah login berhasil
	for {
		conn.Write([]byte(
			"==========================================\n" +
				"Anda dapat memilih fitur-fitur berikut: (Ketik 1-6)\n" +
				"1. Kirim pesan ke semua orang (General Chat)\n" +
				"2. Pilih chatroom untuk mengirim pesan\n" +
				"3. Gabung dengan chatroom\n" +
				"4. Tinggalkan chatroom\n" +
				"5. Buat chatroom baru\n" +
				"6. Keluar dari MariChatting\n" +
				"==========================================\n" + " "))

		pilihan, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected.\n", name)
			s.handleDisconnect(conn)
			return // Keluar dari loop dan goroutine
		}

		pilihan = strings.TrimSpace(pilihan)
		// Jika pilihan adalah 6 (keluar), tangani dan return
		if pilihan == "6" {
			conn.Write([]byte("Anda memilih untuk keluar dari MariChatting.\n"))
			s.broadcast(fmt.Sprintf("\n[PEMBERITAHUAN] %s telah meninggalkan aplikasi MariChatting.\n", s.clients[conn]), conn)
			s.handleDisconnect(conn)
			return // Keluar dari loop dan goroutine
		}

		// Buat WaitGroup untuk menangani pilihan
		wg := sync.WaitGroup{}
		wg.Add(1)
		go s.handlePilihan(conn, pilihan, &wg)
		wg.Wait()
		// Setelah handlePilihan selesai, loop akan berlanjut dan menampilkan menu lagi
	}
}

func (s *ChatServer) handlePilihan(conn net.Conn, pilihan string, wg *sync.WaitGroup) {
	defer wg.Done()
	reader := bufio.NewReader(conn)

	switch pilihan {
	case "1":
		// Langsung masuk ke chatroom "general"
		wgChatroom := sync.WaitGroup{}
		wgChatroom.Add(1)
		go s.handleChatroom(conn, &wgChatroom, "general")
		wgChatroom.Wait()
	case "2":
		// Loop untuk memilih chatroom yang sudah di-join
		for {
			room_joined_message := "Pilih chatroom yang pesannya ingin Anda lihat:\n(Ketik /kembali untuk ke menu utama)\n"
			var joinedRooms []string
			s.mu.Lock()
			for chatroomName := range s.rooms {
				if _, exist := s.rooms[chatroomName][conn]; exist && chatroomName != "general" {
					joinedRooms = append(joinedRooms, chatroomName)
					room_joined_message += fmt.Sprintf("- %s\n", chatroomName)
				}
			}
			s.mu.Unlock()

			if len(joinedRooms) == 0 {
				conn.Write([]byte("Anda belum masuk ke room manapun selain General. (Gabung terlebih dahulu dengan room yang tersedia (Ketik 3))\nTekan enter untuk kembali ke menu utama.\n"))
				_, _ = reader.ReadString('\n')
				return
			}

			conn.Write([]byte(room_joined_message))
			chatroom, _ := reader.ReadString('\n')
			chatroom = strings.TrimSpace(chatroom)

			if chatroom == "/kembali" {
				return
			}
			if chatroom == "" {
				conn.Write([]byte("Nama room tidak boleh kosong!!\n"))
				continue
			}

			s.mu.Lock()
			_, existUser := s.rooms[chatroom][conn]
			s.mu.Unlock()
			if !existUser || chatroom == "general" {
				conn.Write([]byte("Anda belum bergabung atau chatroom tidak valid. Silahkan pilih chatroom yang tersedia.\n"))
				continue
			} else {
				wgChatroom := sync.WaitGroup{}
				wgChatroom.Add(1)
				go s.handleChatroom(conn, &wgChatroom, chatroom)
				wgChatroom.Wait()
				return // Kembali ke menu utama setelah keluar dari room
			}
		}
	case "3":
		// Loop untuk bergabung ke chatroom yang ada
		for {
			room_possible_to_join_message := "Silahkan pilih chatroom yang ingin Anda masuki:\n(Ketik /kembali untuk ke menu utama)\n"
			var possibleRooms []string
			s.mu.Lock()
			for chatroomName := range s.rooms {
				if _, exist := s.rooms[chatroomName][conn]; !exist {
					possibleRooms = append(possibleRooms, chatroomName)
					room_possible_to_join_message += fmt.Sprintf("- %s\n", chatroomName)
				}
			}
			s.mu.Unlock()

			if len(possibleRooms) == 0 {
				conn.Write([]byte("Anda telah memasuki semua chatroom yang tersedia.\nTekan enter untuk kembali ke menu utama.\n"))
				_, _ = reader.ReadString('\n')
				return
			}

			conn.Write([]byte(room_possible_to_join_message))
			chatroom, _ := reader.ReadString('\n')
			chatroom = strings.TrimSpace(chatroom)

			if chatroom == "/kembali" {
				return
			}
			if chatroom == "" {
				conn.Write([]byte("Nama room tidak boleh kosong!!\n"))
				continue
			}

			s.mu.Lock()
			_, roomExists := s.rooms[chatroom]
			_, userInRoom := s.rooms[chatroom][conn]
			s.mu.Unlock()

			if !roomExists {
				conn.Write([]byte(fmt.Sprintf("Tidak ditemukan chatroom dengan nama \"%s\". Silakan coba lagi.\n", chatroom)))
				continue
			}
			if userInRoom {
				conn.Write([]byte("Anda sudah berada di chatroom ini. Silahkan pilih chatroom lain.\n"))
				continue
			}

			s.handleJoinLeaveChatroom(conn, chatroom, "join")
			return // Kembali ke menu utama setelah berhasil join
		}
	case "4":
		// Loop untuk meninggalkan chatroom
		for {
			room_joined_message := "Silahkan pilih chatroom yang ingin Anda tinggalkan:\n(Ketik /kembali untuk ke menu utama)\n"
			var joinedRooms []string
			s.mu.Lock()
			for chatroomName := range s.rooms {
				if _, exist := s.rooms[chatroomName][conn]; exist && chatroomName != "general" {
					joinedRooms = append(joinedRooms, chatroomName)
					room_joined_message += fmt.Sprintf("- %s\n", chatroomName)
				}
			}
			s.mu.Unlock()

			if len(joinedRooms) == 0 {
				conn.Write([]byte("Anda belum masuk ke room apapun untuk ditinggalkan.\nTekan enter untuk kembali ke menu utama.\n"))
				_, _ = reader.ReadString('\n')
				return
			}

			conn.Write([]byte(room_joined_message))
			chatroom, _ := reader.ReadString('\n')
			chatroom = strings.TrimSpace(chatroom)

			if chatroom == "/kembali" {
				return
			}
			if chatroom == "" {
				conn.Write([]byte("Nama chatroom tidak boleh kosong!\n"))
				continue
			}
			if chatroom == "general" {
				conn.Write([]byte("Anda tidak bisa keluar dari room general!!\n"))
				continue
			}

			s.mu.Lock()
			_, existsRoom := s.rooms[chatroom]
			_, existUser := s.rooms[chatroom][conn]
			s.mu.Unlock()

			if !existsRoom || !existUser {
				conn.Write([]byte("Chatroom tidak ditemukan atau Anda bukan member. Silakan coba lagi.\n"))
				continue
			}

			s.handleJoinLeaveChatroom(conn, chatroom, "leave")
			return // Kembali ke menu utama setelah berhasil leave
		}
	case "5":
		// Loop untuk membuat chatroom baru
		for {
			conn.Write([]byte("Silahkan masukkan nama chatroom yang ingin anda buat:\n(Ketik /kembali untuk ke menu utama)\n"))
			chatroomName, _ := reader.ReadString('\n')
			chatroomName = strings.TrimSpace(chatroomName)

			if chatroomName == "/kembali" {
				return
			}
			if chatroomName == "" {
				conn.Write([]byte("Nama chatroom tidak boleh kosong!!\n"))
				continue
			}

			s.mu.Lock()
			_, exists := s.rooms[chatroomName]
			s.mu.Unlock()
			if exists {
				conn.Write([]byte("Chatroom dengan nama tersebut sudah ada. Silakan pilih nama lain.\n"))
				continue
			}

			s.handleCreateChatroom(conn, chatroomName)
			return // Kembali ke menu utama setelah berhasil membuat
		}
	// Case 6 (Keluar) sudah ditangani di loop utama handleClient
	default:
		conn.Write([]byte("Pilihan tidak valid. Silakan coba lagi.\n"))
	}
}

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
