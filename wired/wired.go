package wired

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"
)

var (
	WIRED_SPEC string
)

// There are a few I/O operations we should perform while the server is starting so
// that they aren't repeated for each connection we receive. For instance, reading
// in the Wired specification file for each connection to speed things up later.
func init() {
	// Attempt to read in the Wired specification file.
	file, err := ioutil.ReadFile("wired/WiredSpec_2.0b55.xml")

	// Wired requires the specifications to connect, so we can't continue if an error occurs.
	if err != nil {
		log.Fatalf("Error loading Wired specifications: %s", err.Error())
	}

	WIRED_SPEC = string(file)
}

type Connection struct {
	socket     net.Conn
	serverHost string
	serverPort int
}

func (this *Connection) readData() {
	fmt.Println("Attempting to read data from the socket.")

	result, err := bufio.NewReader(this.socket).ReadString('\r')

	if err != nil {
		log.Panicf("Error reading data from socket: %s", err.Error())
	} else {
		fmt.Println(string(result))
	}

}

func (this *Connection) sendAcknowledgement() {
	fmt.Println("Sending acknowledgement...")

	this.sendTransaction("p7.handshake.acknowledge")
}

//  Responds to a compatibility check from the server.
//
//  Reads in the WiredSpec XML file and sends it to the server. Wired requires that
//  certain characters be encoded before sending. To save processing time the XML
//  should be pre-encoded. To save bandwidth the documentation lines should be removed.
func (this *Connection) sendCompatibilityCheck() {
	fmt.Println("Sending compatibility check...")

	parameters := map[string]string{
		"p7.compatibility_check.specification": WIRED_SPEC,
	}
	this.sendTransaction("p7.compatibility_check.specification", parameters)
	this.readData()
}

// Sends information about the Wired client to the server.
//
// For now this is reporting information about the newest known Mac build.
// In the future, this should report the same information as the Wired version
// that's connecting to the Push server.
func (this *Connection) sendClientInformation() {
	fmt.Println("Sending client information...")

	parameters := map[string]string{
		"wired.info.application.name":    "Wired Client",
		"wired.info.application.version": "2.0",
		"wired.info.application.build":   "268",
		"wired.info.os.name":             "Mac OS X",
		"wired.info.os.version":          "10.8.3",
		"wired.info.arch":                "x86_64",
		"wired.info.supports_rsrc":       "false",
	}

	this.sendTransaction("wired.client_info", parameters)
	this.readData()
}

func (this *Connection) sendTransaction(transaction string, parameters ...map[string]string) {
	// Begin translating the transaction message into XML.
	generatedXML := `<?xml version="1.0" encoding="UTF-8"?>`
	generatedXML += fmt.Sprintf(`<p7:message name="%s" xmlns:p7="http://www.zankasoftware.com/P7/Message">`, transaction)

	// If parameters were sent convert them to XML too.
	if parameters != nil {
		for key, value := range parameters[0] {
			generatedXML += fmt.Sprintf(`<p7:field name="%s">%s</p7:field>`, key, value)
		}
	}

	// End the transaction message.
	// Line break is the end message signal for the socket.
	generatedXML += "</p7:message>\r\n"

	// fmt.Println(generatedXML)

	// Write the data to the socket.
	_, err := this.socket.Write([]byte(generatedXML))

	if err != nil {
		log.Panicf("Error writing data to socket: %s", err.Error())
	}
}

// Sends a users login information to the Wired server.
//
// The password must be converted to a SHA1 digest before sending it to this function.
func (this *Connection) SendLogin(user, password string) {
	fmt.Println("Sending login information...")

	// Send the user login information to the Wired server.
	parameters := map[string]string{
		"wired.user.login":    user,
		"wired.user.password": password,
	}

	this.sendTransaction("wired.send_login", parameters)
	this.readData()
}

func (this *Connection) SetNick(nick string) {
	fmt.Println("Attempting to change nick.")

	parameters := map[string]string{
		"wired.user.nick": nick,
	}

	this.sendTransaction("wired.user.set_nick", parameters)
	this.readData()
}

func (this *Connection) JoinChannel(channel string) {
	fmt.Printf("Attempting to join channel %s.\n", channel)

	// TODO: Reset the user list for this channel.

	// Attempt to join the channel.
	parameters := map[string]string{
		"wired.chat.id": channel,
	}

	this.sendTransaction("wired.chat.join_chat", parameters)
	this.readData()
}

func (this *Connection) ConnectToServer(server string, port int) {
	timeout, _ := time.ParseDuration("15s")

	// Store the connection info so that we can reconnect later if necessary.
	this.serverHost = server
	this.serverPort = port

	// Attempt a socket connection to the server.
	fmt.Println("Beginning socket connection...")
	socket, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", server, port), timeout)
	this.socket = socket

	if err != nil {
		log.Panicf("Connection error: %s\n", err.Error())
	}

	// Start sending Wired connection info.
	fmt.Println("Sending Wired handshake...")
	parameters := map[string]string{
		"p7.handshake.version":          "1.0",
		"p7.handshake.protocol.name":    "Wired",
		"p7.handshake.protocol.version": "2.0",
	}
	this.sendTransaction("p7.handshake.client_handshake", parameters)
	this.readData()

	this.sendAcknowledgement()

	// TODO: Make sure "p7.handshake.compatibility_check" is 1 before sending the compatibility check.
	this.sendCompatibilityCheck()

	// TODO: Make sure "p7.compatibility_check.status" is 1.
	// If it's not, we need to end the cancel the connection.
	this.sendClientInformation()

	this.SendLogin("guest", "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	// TODO: We need to check and see if the login information was correct.
	this.SetNick("Wired APNS")

	// this.JoinChannel("1")

	// Close the socket connection.
	this.socket.Close()
}
