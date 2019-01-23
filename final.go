package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type Client struct {
	name string
	id   int
}

func removeTrailingNewline(s string) string {
	l := len(s)
	if s[l-1] == '\n' {
		return s[:l-1]
	} else {
		return s
	}
}

func main() {
	clientCount := 0
	// int serves as unique id
	allClients := make(map[net.Conn]Client)
	// TCP will push new connections to it
	newConnections := make(chan net.Conn)
	// We will remove those clients from allClients
	deadConnections := make(chan net.Conn)
	// Channel which will contain message from connected clients
	messages := make(chan string)
	// Start TCP server
	server, err := net.Listen("tcp", ":6060")
	if err != nil {
		err = fmt.Errorf("Error launching the server :", err)
		fmt.Println(err)
	}

	// Server accepts connections forever and pushes new ones
	// to the dedicated channel
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				fmt.Println(err)
			}
			newConnections <- conn
		}

	}()

	// Infinite loop
	for {
		select {

		// Continuously accept new clients
		case conn := <-newConnections:
			log.Printf("Accepted new client with id %d", clientCount)
			allClients[conn] = Client{id: clientCount}
			clientCount++
			// Read all incoming messages from this client into a goroutine
			// and push them to the message chan
			go func(conn net.Conn) {
				client := allClients[conn]
				reader := bufio.NewReader(conn)
				conn.Write([]byte("Welcome to the server ! \n"))
				conn.Write([]byte("Please enter the pseudo you want to use : "))
				pseudo, _ := reader.ReadString('\n')
				pseudo = removeTrailingNewline(pseudo)

				log.Printf("%t", strings.ContainsAny(pseudo, " ,;:?!/"))
				for strings.ContainsAny(pseudo, " ,;:?/.§%") {
					conn.Write([]byte(`Pseudo should not contain any of the following characters : " ' ,;:/.?%'`))
					conn.Write([]byte("\n Please enter a new pseudo : "))
					pseudo, _ = reader.ReadString('\n')
					pseudo = removeTrailingNewline(pseudo)
				}
				client.name = pseudo
				reader = bufio.NewReader(conn)
				for {
					incoming, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					messages <- fmt.Sprintf("%s > %s", client.name, incoming)
				}
				// When encounter an error, the client will be removed
				deadConnections <- conn

			}(conn)


		// Continuously read incoming messages and broadcast them
		case message := <-messages:
			for conn := range allClients {
				//Send the message in a go routine
				go func(conn net.Conn, message string) {
					_, err := conn.Write([]byte(message))
					// If it doesn't work the connection is dead
					if err != nil {
						deadConnections <- conn
					}
				}(conn, message)
			}
			log.Printf("New message : %s", message)

		//Remove dead clients
		case conn := <-deadConnections:
			log.Printf("Client %d disconnected", allClients[conn].id)
			delete(allClients, conn)
		}

	}

}