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
	fmt.Println("Telah tersambung ke server!")

	// Fungsi goroutine untuk membaca pesan dari server

	connectionReader := bufio.NewReader(conn)
	reader := bufio.NewReader(os.Stdin)

	// Mengambil nama pengguna dari input dan mengirim ke server

	for {
		fmt.Print("Masukkan nama pengguna: \n")
		name, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("Cannot read your input, please try again!")
			continue
		}
		name = strings.TrimSpace(name)

		conn.Write([]byte(name + "\n"))

		//Untuk mengetahui jika nama valid, ambil response dari server dan cek jika ada balasan dari server
		msg, _ := connectionReader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		fmt.Printf("%s\n", msg)
		if strings.Contains(msg, "Selamat datang di MariChatting") {
			break
		} else {
			continue
		}
	}
	go readMessages(conn)

	// Loop untuk mengirim pesan atau perintah ke server
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
