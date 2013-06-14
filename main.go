package main

import (
	"fmt"
	"github.com/mattprice/Wired-APNS/wired"
	"log"
	"runtime"
)

func main() {
	log.Println("*** Starting server ***")

	// Tell Go to use the maximum number of CPU threads.
	runtime.GOMAXPROCS(runtime.NumCPU())

	connection := &wired.Connection{
		Host: "chat.embercode.com",
		Port: 2359,
	}
	connection.Connect()

	// Wait for user input before disconnecting from the server.
	var input string
	fmt.Scanln(&input)
	connection.Disconnect()

	log.Println("*** Exiting server ***")
}
