package main

import (
	"github.com/mattprice/Wired-APNS/wired"
)

func main() {
	connection := new(wired.Connection)
	connection.ConnectToServer("chat.embercode.com", 2359)
}
