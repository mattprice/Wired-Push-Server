package main

import (
	"fmt"
	"github.com/mattprice/Wired-APNS/wired"
	"log"
	"runtime"
)

func main() {
	// Tell Go to use the maximum number of CPU threads.
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("*** Starting server ***")

	connection := new(wired.Connection)
	connection.ConnectToServer("chat.embercode.com", 2359)

	// Wait for user input before disconnecting from the server.
	var input string
	fmt.Scanln(&input)

	connection.Disconnect()
	log.Println("*** Exiting server ***")
}
