package main

import (
	"fmt"
	"github.com/mattprice/Wired-APNS/wired"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	connection := new(wired.Connection)
	connection.ConnectToServer("chat.embercode.com", 2359)

	// Wait for user input before disconnecting from the server.
	var input string
	fmt.Scanln(&input)

	connection.Disconnect()
}
