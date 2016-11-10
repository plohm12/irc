package main

import(
	"fmt"
	irc "ircserver/ircdefs"
	"ircserver/parser"
	"log"
	"net"
	"strings"
)

var(
	password string = ""
)

func handlePass(msg *parser.Message) (err string) {
	err = ""
	if msg.Params.Num > 0 {
		if password != "" {
			err = fmt.Sprint(irc.ERR_ALREADYREGISTRED)
		} else {
			password = msg.Params.Others[0]
		}
	} else {
		err = fmt.Sprint(irc.ERR_NEEDMOREPARAMS)
	}
	return
}

/* generic message handler */
func handleMessage(msg *parser.Message) (reply string) {
	reply = ""
	switch strings.ToUpper(msg.Command) {
	case "QUIT":
		// unregister this user's info
		reply = irc.ERR_CONNCLOSED
	case "PASS":
		reply = handlePass(msg)
	case "NICK":
		//handle nick
	case "USER":
		//handle user
	}
	return
}

// Given a connection, read and print messages to console
//TODO: after parsing message, check state info to determine if connection should stay open
func serve(conn net.Conn) {
	//buf := make([]byte, 512)
	p := parser.NewParser(conn)
	defer conn.Close()

	// Accept PASS registration message
	for {
		fmt.Println("Waiting for a PASS message...")
		if msg, err := p.Parse(); err != nil {
			log.Println(err)
			_, _ = conn.Write([]byte(irc.ERR_GENERAL))
		} else {
			if strings.ToUpper(msg.Command) == "PASS" {
				if err := handlePass(msg); err != "" {
					conn.Write([]byte(err))
				} else {
					break;
				}
			}
		}
	}


	// Accept NICK registration message


	// Accept USER registration message

	for {
		//n, err := conn.Read(buf)
		if msg, err := p.Parse(); err != nil {
			log.Println(err)
			_, _ = conn.Write([]byte(irc.ERR_GENERAL))
			// conn.Close()
			// return
		} else {
			parser.Print(msg)
			reply := handleMessage(msg)
			_, _ = conn.Write([]byte(reply))
		}
	}
}

// Program entry point
func main() {
	// Listen for TCP connections on this address and port
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalln(err)
	}

	// Accept and serve each connection in a new goroutine
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go serve(conn)
	}
}
