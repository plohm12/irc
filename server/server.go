package main

import(
	"fmt"
	"irc"
	"irc/parser"
	"log"
	"net"
	"strconv"
	"strings"
)

var(
	password string = ""
	nick string = ""
	username string = ""
	mode int = 0
	realname string = ""
)

func handlePass(msg *parser.Message) (reply string) {
	reply = ""
	if msg.Params.Num > 0 {
		if password != "" {
			reply = irc.ERR_ALREADYREGISTRED
		} else {
			password = msg.Params.Others[0]
		}
	} else {
		reply = irc.ERR_NEEDMOREPARAMS
	}
	return
}

func handleNick(msg *parser.Message) (reply string) {
	reply = ""
	if password == "" {
		reply = irc.ERR_NOTREGISTERED
		return
	}
	if msg.Params.Num < 1 {
		reply = irc.ERR_NONICKNAMEGIVEN
		return
	}
	//TODO: check that nick fits spec
	//TODO: check for collisions
	nick = msg.Params.Others[0]
	return
}

func handleUser(msg *parser.Message) (reply string) {
	reply = ""
	if password == "" || nick == "" {
		reply = irc.ERR_NOTREGISTERED
		return
	}
	if msg.Params.Num < 4 {
		reply = irc.ERR_NEEDMOREPARAMS
		return
	}
	if username != "" {
		reply = irc.ERR_ALREADYREGISTRED
		return
	}

	//TODO: probably check that each field is safe
	username = msg.Params.Others[0]
	if m, err := strconv.Atoi(msg.Params.Others[1]); err != nil {
		log.Println(err)
		username = ""
		mode = 0
		reply = irc.ERR_GENERAL
		return
	} else {
		mode = m
	}
	// discard 3rd param; it is unused
	realname = msg.Params.Others[3]
	reply = irc.RPL_WELCOME
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
		reply = handleNick(msg)
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
	fmt.Println("A connection was opened.")

	// Accept PASS registration message
	// for {
	// 	fmt.Println("Waiting for a PASS message...")
	// 	if msg, err := p.Parse(); err != nil {
	// 		log.Println(err)
	// 		_, _ = conn.Write([]byte(irc.ERR_GENERAL))
	// 	} else {
	// 		parser.Print(msg)
	// 		if strings.ToUpper(msg.Command) == "PASS" {
	// 			if err := handlePass(msg); err != "" {
	// 				conn.Write([]byte(err))
	// 			} else {
	// 				break;
	// 			}
	// 		}
	// 	}
	// }


	// Accept NICK registration message


	// Accept USER registration message

	for {
		//n, err := conn.Read(buf)
		if msg, err := p.Parse(); err != nil {
			log.Println(err)
			_, _ = conn.Write([]byte(irc.ERR_GENERAL))
			conn.Close()
			return
		} else {
			parser.Print(msg)
			reply := handleMessage(msg)
			_, _ = conn.Write([]byte(reply))
			if reply == irc.ERR_CONNCLOSED {
				conn.Close()
				fmt.Println("A connection was closed.")
				return
			}
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
