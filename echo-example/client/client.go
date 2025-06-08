package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Membuka koneksi ke server yang berjalan di localhost:9090
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!\n")
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("Connected to server!")

	// Fungsi goroutine untuk membaca pesan dari server
	go readMessages(conn)

	reader := bufio.NewReader(os.Stdin)

	// Mengambil nama pengguna dari input dan mengirim ke server
	fmt.Print("Enter your name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	// Mengirim nama pengguna ke server
	conn.Write([]byte(name + "\n"))

	// Loop untuk membaca dan mengirim pesan
	for {
		// fmt.Printf("[%s] : ", name)
		message, _ := reader.ReadString('\n')
		conn.Write([]byte(message)) // Kirim pesan ke server
	}
}

// Fungsi ini berjalan di goroutine terpisah untuk terus membaca pesan dari server
func readMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		// Membaca satu baris pesan dari server
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from server.")
			os.Exit(0)
		}
		fmt.Print(message) // Tampilkan pesan dari server
	}
}
