package main

import (
	"github.com/mattprice/Wired-APNS/wired"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	connection := new(wired.Connection)
	connection.ConnectToServer("chat.embercode.com", 2359)
}
