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
			os.Exit(1)
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

	fmt.Printf("New client	 connected: %s\n", name)
	s.broadcast(fmt.Sprintf("%s telah masuk ke aplikasi MariChatting.\n", name), conn)

	for {
		conn.Write([]byte(
			"==========================================\n" +
				"Anda dapat memilih fitur-fitur berikut:\n" +
				"1. Kirim pesan ke semua orang\n" +
				"2. Pilih chatroom\n" +
				"3. Gabung dengan chatroom\n" +
				"4. Tinggalkan chatroom\n" +
				"5. Buat chatroom baru\n" +
				"6. Keluar dari MariChatting\n" +
				"==========================================\n"))

		pilihan, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected.\n", name)
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			return
		} else {
			// Menghapus newline dan spasi dari pilihan
			pilihan = strings.TrimSpace(pilihan)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go s.handlePilihan(conn, pilihan, &wg)
			wg.Wait()
			continue
		}
	}
}

func (s *ChatServer) handlePilihan(conn net.Conn, pilihan string, wg *sync.WaitGroup) {
	defer wg.Done()
	switch pilihan {
	case "1":
		wgChatroom := sync.WaitGroup{}
		wgChatroom.Add(1)
		go s.handleChatroom(conn, &wgChatroom, "general")
		wgChatroom.Wait()
	case "2":
		for {
			room_joined_message := "Chatroom yang tersedia:\n"
			room_joined := 0
			reader := bufio.NewReader(conn)

			for chatroomName := range s.rooms {
				_, exist := s.rooms[chatroomName][conn]
				if chatroomName == "general" || !exist {
					continue
				}
				room_joined += 1
				room_joined_message += fmt.Sprintf("%s\n", chatroomName)

			}

			//Kalo belum masuk ke room manapun
			if room_joined > 0 {
				conn.Write([]byte(room_joined_message))
			} else {
				conn.Write([]byte("Anda belum masuk ke room apapun, silakan pilih opsi yang lain.\nTekan enter untuk melanjutkan\n"))
				_, _ = reader.ReadString('\n')
				break
			}

			chatroom, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read chatroom in case 2!\n")
				os.Exit(1)
			}
			chatroom = strings.TrimSpace(chatroom)

			if chatroom == "" {
				conn.Write([]byte("Nama room tidak boleh kosong!!\n"))
				continue
			}

			_, existRoom := s.rooms[chatroom]
			_, existUser := s.rooms[chatroom][conn]
			if !existRoom || !existUser {
				conn.Write([]byte("Chatroom tidak ditemukan. Silahkan pilih chatroom yang tersedia.\n"))
				continue
			} else {
				wgChatroom := sync.WaitGroup{}
				wgChatroom.Add(1)
				go s.handleChatroom(conn, &wgChatroom, chatroom)
				wgChatroom.Wait()
				break
			}
		}
	case "3":
		for {
			room_possible_to_join_message := "Silahkan pilih chatroom yang ingin Anda masuki:\n"
			room_possible_to_join := 0
			reader := bufio.NewReader(conn)

			for chatroomName := range s.rooms {
				_, exist := s.rooms[chatroomName][conn]
				if chatroomName == "general" || exist {
					continue
				}
				room_possible_to_join += 1
				room_possible_to_join_message += fmt.Sprintf("%s\n", chatroomName)
			}

			if room_possible_to_join > 0 {
				conn.Write([]byte(room_possible_to_join_message))
			} else {
				conn.Write([]byte("Anda telah memasuki semua chatroom yang tersedia.\nTekan enter untuk melanjutkan\n"))
				_, _ = reader.ReadString('\n')
				break
			}

			chatroom, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			chatroom = strings.TrimSpace(chatroom)
			if chatroom == "" {
				conn.Write([]byte("Nama room tidak boleh kosong!!\n"))
				continue
			}

			if _, exist := s.rooms[chatroom][conn]; exist {
				conn.Write([]byte("Anda sudah berada di chatroom ini. Silahkan pilih chatroom lain.\n"))
				continue
			} else {
				if _, exists := s.rooms[chatroom]; exists {
					wgJoinLeave := sync.WaitGroup{}
					wgJoinLeave.Add(1)
					go s.handleJoinLeaveChatroom(conn, &wgJoinLeave, chatroom, "join")
					wgJoinLeave.Wait()
					break
				} else {
					conn.Write([]byte(fmt.Sprintf("Tidak ditemukkan chatroom dengan nama \"%s\".Silakan coba lagi.\n", chatroom)))
					continue
				}
			}
		}
	case "4":
		for {
			room_joined_message := "Silahkan pilih chatroom yang ingin Anda tinggalkan:\n"
			room_joined := 0
			reader := bufio.NewReader(conn)

			for chatroomName := range s.rooms {
				_, exist := s.rooms[chatroomName][conn]
				if chatroomName == "general" || !exist {
					continue
				}
				room_joined += 1
				room_joined_message += fmt.Sprintf("%s\n", chatroomName)
			}

			//Kalo belum masuk ke room manapun selain general
			if room_joined > 0 {
				conn.Write([]byte(room_joined_message))
			} else {
				conn.Write([]byte("Anda belum masuk ke room apapun, silakan pilih opsi yang lain.\nTekan enter untuk melanjutkan\n"))
				_, _ = reader.ReadString('\n')
				break
			}
			chatroom, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			chatroom = strings.TrimSpace(chatroom)

			if chatroom == "" {
				conn.Write([]byte("Anda tidak mengisi nama chatroom yang ingin Anda tinggalkan!!\n"))
				continue
			} else if chatroom == "general" {
				conn.Write([]byte("Anda tidak bisa keluar dari room general!!\n"))
				continue

			}

			_, existsRoom := s.rooms[chatroom]
			_, existUser := s.rooms[chatroom][conn]
			if !existsRoom || !existUser {
				conn.Write([]byte("Chatroom tidak ditemukan. Silakan coba lagi.\n"))
				continue
			} else {
				wgJoinLeave := sync.WaitGroup{}
				wgJoinLeave.Add(1)
				go s.handleJoinLeaveChatroom(conn, &wgJoinLeave, chatroom, "leave")
				wgJoinLeave.Wait()
				break
			}

		}
	case "5":
		for {
			conn.Write([]byte("Silahkan masukkan nama chatroom yang ingin anda buat.\n"))
			reader := bufio.NewReader(conn)
			chatroomName, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read chatroomName in case 5!\n")
				os.Exit(1)
			}

			chatroomName = strings.TrimSpace(chatroomName)

			if chatroomName == "" {
				conn.Write([]byte("Nama chatroom tidak boleh kosong!!\n"))
				continue
			}

			if _, exists := s.rooms[chatroomName]; exists {
				conn.Write([]byte("Chatroom dengan nama tersebut sudah ada. Silakan pilih nama lain.\n"))
				continue
			} else {
				wg := sync.WaitGroup{}
				wg.Add(1)
				go s.handleCreateChatroom(conn, &wg, chatroomName)
				wg.Wait()
				break
			}
		}
	case "6":
		conn.Write([]byte("Anda memilih untuk keluar dari MariChatting.\n"))
		s.mu.Lock()
		for roomName := range s.rooms {
			delete(s.rooms[roomName], conn)
		}
		s.mu.Unlock()
		s.broadcast(fmt.Sprintf("%s telah meninggalkan MariChatting.\n", s.clients[conn]), conn)
		conn.Close()
		return
	default:
		conn.Write([]byte("Pilihan tidak valid. Silakan coba lagi.\n"))
		return
	}
}

func (s *ChatServer) handleJoinLeaveChatroom(conn net.Conn, wg *sync.WaitGroup, chatroom string, operasi string) {
	defer wg.Done()

	switch operasi {
	case "join":
		s.mu.Lock()
		s.rooms[chatroom][conn] = false
		dummy := s.clients[conn]
		s.mu.Unlock()
		conn.Write([]byte(fmt.Sprintf("Anda telah bergabung dalam chatroom %s.\n", chatroom)))
		s.broadcastPerRoom(fmt.Sprintf("%s telah bergabung dalam chatroom %s.\n", dummy, chatroom), chatroom, conn)
	case "leave":
		s.mu.Lock()
		delete(s.rooms[chatroom], conn)
		dummy := s.clients[conn]
		s.mu.Unlock()
		conn.Write([]byte(fmt.Sprintf("Anda telah meninggalkan chatroom %s.\n", chatroom)))
		s.broadcastPerRoom(fmt.Sprintf("%s tidak lagi tergabung dalam chatroom %s.\n", dummy, chatroom), chatroom, conn)
	}
}

func (s *ChatServer) handleChatroom(conn net.Conn, wg *sync.WaitGroup, roomName string) {
	defer wg.Done()
	clientName := s.clients[conn]

	s.mu.Lock()
	s.rooms[roomName][conn] = true
	s.mu.Unlock()

	reader := bufio.NewReader(conn)
	conn.Write([]byte(fmt.Sprintf("Selamat datang di room %s! Silahkan ketik pesan Anda!\nKetik '/exit' untuk keluar dari room.\n", roomName)))
	s.broadcastPerRoom(fmt.Sprintf("%s telah masuk ke chatroom %s.\n", clientName, roomName), roomName, conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		message = strings.TrimSpace(message)

		if message == "/exit" {
			s.mu.Lock()
			s.rooms[roomName][conn] = false
			s.mu.Unlock()
			conn.Write([]byte(fmt.Sprintf("Anda telah keluar dari chatroom %s.\n", roomName)))
			s.broadcastPerRoom(fmt.Sprintf("%s telah keluar dari chatroom %s.\n", clientName, roomName), roomName, conn)
			break
		}

		s.broadcastPerRoom(fmt.Sprintf("[%s]: %s", clientName, message), roomName, conn)
	}
}

func (s *ChatServer) broadcastPerRoom(message string, roomName string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.rooms[roomName] {
		if conn != sender && s.rooms[roomName][conn] {
			_, err := conn.Write([]byte(message + "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send message to %s: %v\n", s.clients[conn], err)
			}
		}
	}
}

// Method receiver untuk broadcast
func (s *ChatServer) broadcast(message string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.clients {
		if conn != sender {
			_, err := conn.Write([]byte(message + "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send message to %s: %v\n", s.clients[conn], err)
			}
		}
	}
}

func (s *ChatServer) handleCreateChatroom(conn net.Conn, wg *sync.WaitGroup, chatroomName string) {
	defer wg.Done()

	s.mu.Lock()
	s.rooms[chatroomName] = make(map[net.Conn]bool)
	s.rooms[chatroomName][conn] = false
	clientName := s.clients[conn]
	s.mu.Unlock()

	conn.Write([]byte(fmt.Sprintf("Chatroom %s telah berhasil dibuat.\n", chatroomName)))
	s.broadcast(fmt.Sprintf("%s telah membuat chatroom baru: %s", clientName, chatroomName), conn)
}

// Mengecek apakah nama sudah dipakai client lain
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
