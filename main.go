package main

import (
	"fmt"
	"github.com/mattprice/Wired-Push-Server/wired"
	"log"
	"runtime"
)

func main() {
	log.Println("*** Starting server ***")

	// Tell Go to use the maximum number of CPU threads.
	runtime.GOMAXPROCS(runtime.NumCPU())

	conn := wired.Connect("chat.embercode.com", 2359)

	// Wait for user input before disconnecting from the server.
	var input string
	fmt.Scanln(&input)
	conn.Disconnect()

	log.Println("*** Exiting server ***")
}
