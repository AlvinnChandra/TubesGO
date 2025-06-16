package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

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
