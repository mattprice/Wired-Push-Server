package wired

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"
)

var WIRED_SPEC string

// There are a few I/O operations we should perform while the server is starting so
// that they aren't repeated for each connection we receive. For instance, reading
// in the Wired specification file to speed things up.
func init() {
	file, err := ioutil.ReadFile("wired/WiredSpec_2.0b55.xml")
	if err != nil {
		// We can't continue since Wired requires the specifications to connect.
		log.Fatalf("Error loading Wired specifications: %v", err)
	}

	WIRED_SPEC = string(file)
}

type Connection struct {
	socket net.Conn

	serverHost string
	serverPort int

	userID string
}

// Connects to the specified server and port.
func (this *Connection) ConnectToServer(server string, port int) {
	timeout, _ := time.ParseDuration("15s")

	// Store the connection info so that we can reconnect later if necessary.
	this.serverHost = server
	this.serverPort = port

	// Attempt a socket connection to the server.
	log.Println("Beginning socket connection...")
	socket, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", server, port), timeout)
	this.socket = socket

	if err != nil {
		log.Panicf("Connection error: %v\n", err)
	}

	// Start sending Wired connection info.
	log.Println("Sending Wired handshake...")
	parameters := map[string]string{
		"p7.handshake.version":          "1.0",
		"p7.handshake.protocol.name":    "Wired",
		"p7.handshake.protocol.version": "2.0",
	}
	this.sendTransaction("p7.handshake.client_handshake", parameters)

	// Start listening for server responses.
	go this.readData()
}

// Disconnects from the server.
func (this *Connection) Disconnect() {
	log.Println("Disconnecting from server...")

	// Alert the Wired server that we're disconnecting.
	parameters := map[string]string{
		"wired.user.id":                 this.userID,
		"wired.user.disconnect_message": "",
	}
	this.sendTransaction("wired.user.disconnect_user", parameters)

	// Close the socket connection.
	this.socket.Close()
}

// Sends a user's login information to the Wired server.
//
// The password must be converted to a SHA1 digest before sending it to this function.
func (this *Connection) SendLogin(user, password string) {
	log.Println("Sending login information...")

	// Send the user login information to the Wired server.
	parameters := map[string]string{
		"wired.user.login":    user,
		"wired.user.password": password,
	}
	this.sendTransaction("wired.send_login", parameters)
}

// Sets a user's nickname.
func (this *Connection) SetNick(nick string) {
	log.Println("Attempting to change nick...")

	parameters := map[string]string{
		"wired.user.nick": nick,
	}
	this.sendTransaction("wired.user.set_nick", parameters)
}

// Sets a user's status.
func (this *Connection) SetStatus(status string) {
	log.Println("Attempting to change status...")

	parameters := map[string]string{
		"wired.user.status": status,
	}
	this.sendTransaction("wired.user.set_status", parameters)
}

// Sets a user's avatar.
func (this *Connection) SetIcon(icon string) {
	log.Println("Attempting to change icon...")

	parameters := map[string]string{
		"wired.user.icon": icon,
	}
	this.sendTransaction("wired.user.set_icon", parameters)
}

// Joins the specified channel.
//
// Under most circumstances users will only ever join channel 1, the public channel.
func (this *Connection) JoinChannel(channel string) {
	log.Printf("Attempting to join channel %s...\n", channel)

	// Attempt to join the channel.
	parameters := map[string]string{
		"wired.chat.id": channel,
	}
	this.sendTransaction("wired.chat.join_chat", parameters)
}

// Sends an acknowledgement to the Wired server.
func (this *Connection) sendAcknowledgement() {
	log.Println("Sending acknowledgement...")

	this.sendTransaction("p7.handshake.acknowledge")
}

// Replies to a ping request from the Wired server.
func (this *Connection) sendPingReply() {
	// log.Println("Attempting to send ping reply...")

	this.sendTransaction("wired.ping")
}

//  Responds to a compatibility check from the server.
//
//  Reads in the WiredSpec XML file and sends it to the server. Wired requires that
//  certain characters be encoded before sending. To save processing time the XML
//  should be pre-encoded. To save bandwidth the documentation lines should be removed.
func (this *Connection) sendCompatibilityCheck() {
	log.Println("Sending compatibility check...")

	parameters := map[string]string{
		"p7.compatibility_check.specification": WIRED_SPEC,
	}
	this.sendTransaction("p7.compatibility_check.specification", parameters)
}

// Sends information about the Wired client to the server.
//
// For now this is reporting information about the newest known Mac build.
// In the future, this should report the same information as the Wired version
// that's connecting to the Push server.
func (this *Connection) sendClientInformation() {
	log.Println("Sending client information...")

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
}

// Sends a transaction to the Wired server.
//
// All transactions required a transaction name, but the parameters map is optional.
// Only the first parameters map is read. Multiple parameter maps will be ignored.
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

	// Write the data to the socket.
	_, err := this.socket.Write([]byte(generatedXML))

	if err != nil {
		log.Println("Error writing data to socket: %v", err)
	}
}

// Reads data from the socket and then passes it off for processing.
//
// Until the socket disconnects, we could receive data from the Wired server at
// any time. To make sure we don't miss any messages, readData will loop forever
// in its own Goroutine until it recieves data and then immediatley pass it off
// to another Goroutine for processing.
func (this *Connection) readData() {
	for {
		// log.Println("Attempting to read data from the socket.")

		data, err := bufio.NewReader(this.socket).ReadBytes('\r')
		if err != nil {
			log.Println("Error reading data from socket: %v", err)
			continue
		}

		go this.processData(&data)
	}
}

func (this *Connection) processData(data *[]byte) {
	type p7Field struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",innerxml"`
	}

	type p7Message struct {
		Name   string    `xml:"name,attr"`
		Fields []p7Field `xml:"field"`
	}

	// Decode the XML document.
	message := new(p7Message)
	err := xml.Unmarshal(*data, &message)
	if err != nil {
		log.Printf("Error decoding XML document: %v\n%v", err, string(*data))
		return
	}

	if message.Name == "p7.handshake.server_handshake" {
		// Server Handshake
		log.Println("Received handshake.")

		go this.sendAcknowledgement()

		for _, field := range message.Fields {
			if field.Name == "p7.handshake.compatibility_check" {
				if field.Value == "1" {
					go this.sendCompatibilityCheck()
				} else {
					go this.sendClientInformation()
				}
			}
		}
	} else if message.Name == "p7.compatibility_check.status" {
		// Compatibility Check
		log.Println("Received compatibility status.")

		for _, field := range message.Fields {
			if field.Name == "p7.compatibility_check.status" {
				if field.Value == "1" {
					go this.sendClientInformation()
				} else {
					// TODO: Panic will crash the entire server right now.
					// We need to do some defer()'s and recover()'s in the main Goroutine
					// so that only this Connection closes.
					log.Panic("Compatibility mismatch.")
				}
			}
		}
	} else if message.Name == "wired.server_info" {
		// Server Info
		log.Println("Received server info.")

		// If the APNS is connecting by itself then we need to log in.
		go this.SendLogin("guest", "da39a3ee5e6b4b0d3255bfef95601890afd80709")
	} else if message.Name == "wired.login" {
		// Login Successful
		log.Println("Login was successful.")

		for _, field := range message.Fields {
			if field.Name == "wired.user.id" {
				this.userID = field.Value
			}
		}

		go func() {
			this.SetNick("Applejack")
			this.SetStatus("Wired APNs Test")
			this.SetIcon(`iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAMAAACdt4HsAAAAV1BMVEW6X
				0SuaEWmb0ZwpU3bSEDrPj/sPj7qQD/pQD9evFBWwlFevFBgulBJzVNavlFgulBYwFHxO
				j7qQD/8MDzqQD/qQD/qPz/6MD3qQD/sPj77KjxgulDqQD8msS8AAAAAG3RSTlMAAwsaF
				ShAZ1U3WH+lqMri9f7w9dzNua2fhWDaFcMCAAACUklEQVR4XsWW67aiMAyFd0qvwOFmw
				WJ4/+ecg46OnWOh4po1309rdmJSu4N/S1kVhCM4cYtv2OEAVHMjAFTMxwTWwJrg+JsCB
				3D1NXXDzC2OUTJX4G8aHFZoHDN3JaDneVYHFNqKuavoMi4rAe8grgqlKwtx7pcbE7Kxk
				/dBA0SkTssDg0zMLeCsgEu//GEgZKEeAVovERZZzEuKmZDDmAjPbaMYkgIjIQOZFvAfV
				4As/Ic9QEhPIfcepdDIQqTiB+Czm2QIedDrOXhCLvqlgEI+9kUDNN5Bj0tMkHgTMw3Xx
				FOYpmDi8os8t5JKayXwk4K5e5jFlhIlDouWuf5tH7VABDVtLbADiY47ARQ1c0uIEMzcU
				I7zuWshXCOm6NbPNpBaAqC773GJGNEyb3RYhHUuCvhiOH5VLbVbZir9cuUCx2snu6aIg
				mtHaLjOeG4v+Cqbr8rF+d3N0KuMC95rIghtzudgo9a2VHGZ89adpD6fTn1smfXaVMcu6
				z/a9z8funLtannfaLTRmY7ln0fI1e0iGr8WJ7Ica8CdomGuSwHI6YUP0LArAJRdJwjKv
				7JjteVXcp78OM2aUNKzt405AgHQ95DTLGNnks9DSGAjO+3PU2Ql+xUMAvOSxNKua8+EM
				S1gaHeMEhg2BHZN12BTwO7ZzYhtAUXbV3GQ94Nkh59QqQQhdyWxCbvTGz2MUVGuUdLeV
				kT4G2mtuX3dG3ocm7c2CiI7h1lHvuXfXwsJyf7GI8LhzWxUwEGFYbUfA+DYTuKtUEoS3
				seG0Y/BEj6BCP+BX0F2mxFLbI8LAAAAAElFTkSuQmCC`)

			this.JoinChannel("1")
		}()
	} else if message.Name == "wired.send_ping" {
		// Ping Request
		go this.sendPingReply()
	} else if message.Name == "wired.error" {
		// Wired Errors
		for _, field := range message.Fields {
			if field.Value == "wired.error.login_failed" {
				log.Panicln("Login failed:", "Username or password is incorrect.")
			} else if field.Value == "wired.banned" {
				log.Panicln("Login failed:", "User is banned from this server.")
			} else {
				log.Println("*** ERROR:", field.Value, "***")
			}
		}
	} else {
		// log.Printf("%q\n", message.Name)
		// for _, field := range message.Fields {
		// 	log.Printf("  %q => %q\n", field.Name, field.Value)
		// }
	}
}
