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
}

func main() {
	// Inisialisasi ChatServer
	server := &ChatServer{
		clients: make(map[net.Conn]string),
	}

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

	conn.Write([]byte("Enter your name: "))
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	//Memasukkan nama client ke dalam map
	s.mu.Lock()
	s.clients[conn] = name
	s.mu.Unlock()

	fmt.Printf("New client connected: %s\n", name)
	s.broadcast(fmt.Sprintf("%s has joined the chat.\n", name), conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected.\n", name)
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			s.broadcast(fmt.Sprintf("%s has left the chat.\n", name), conn)
			return
		}

		s.broadcast(fmt.Sprintf("%s: %s", name, message), conn)
	}
}

// Method receiver untuk broadcast
func (s *ChatServer) broadcast(message string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.clients {
		if conn != sender {
			conn.Write([]byte(message))
		}
	}
}
