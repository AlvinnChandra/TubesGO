package main

import (
	"fmt"
	"net"
	"os"
	"sync"
)

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
