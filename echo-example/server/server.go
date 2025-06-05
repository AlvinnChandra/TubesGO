package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)


var (
	// Map untuk menyimpan koneksi dan nama client
	clients = make(map[net.Conn]string)

	//Mutex untuk sinkronisasi akses map
	mutex = &sync.Mutex{}
)

func main() {
	// Membuka server TCP di port 9090
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server listening on :9090...")

	//Pada bagian ini, digunakan untuk menerima koneksi client
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			continue
		}

		//Memproses koneksi client dalam goroutine
		go handleClient(conn)
	}
}

// Untuk komunikasi dengan Client
func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	//Mengambil nama client
	conn.Write([]byte("Enter your name: "))
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	//Mengunci akses variable bersama, saat sedang digunakan oleh goroutine
	mutex.Lock()
	clients[conn] = name
	mutex.Unlock()

	fmt.Printf("New client connected: %s\n", name)

	//Broadcat pesan ke semua client yang lain
	broadcast(fmt.Sprintf("%s has joined the chat.\n", name), conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected.\n", name)
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			broadcast(fmt.Sprintf("%s has left the chat.\n", name), conn)
			return
		}

		broadcast(fmt.Sprintf("%s: %s", name, message), conn)
	}
}

// Mengirimankan pesan ke semua client lain
func broadcast(message string, sender net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn := range clients {
		if conn != sender { // Jangan mengirim pesan ke pengirim (diri sendiri)
			conn.Write([]byte(message))
		}
	}
}
