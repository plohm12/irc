//TODO: add better closure defer function
//TODO: make client more user-friendly
//TODO: add parser
//TODO: enumerate help messages
//TODO: add golang channel for response handling

package main

import (
	"bufio"
	//"bytes"
	"fmt"
	"irc"
	"log"
	"net"
	"os"
	"strings"
)

var (
	conn     net.Conn
	reader   *bufio.Reader
	nick     string = ""
	password string = ""
	user     string = ""
	mode     int    = 0
	realname string = ""
)

func printWelcome() {
	fmt.Println("Welcome to the Internet Relay Chat!")
	fmt.Println("For HELP, type \"help\".")
	fmt.Println("To terminate your session, type \"quit\".")
}

func printHelp() {
	fmt.Println("Enter: PASS [password]")
}

func register() {
	for password == "" {
		fmt.Print("Create a session password:\n> ")
		s := getInput()
		if strings.ToUpper(s) == "HELP" {
			continue
		}
		sendMsg("PASS", s)
		//		if handleResponse() {
		//			fmt.Println("Password success!")
		//			password = s
		//		}
		password = s
	}
	for nick == "" {
		fmt.Print("Enter your nickname:\n> ")
		s := getInput()
		if strings.ToUpper(s) == "HELP" {
			continue
		}
		sendMsg("NICK", s)
		// handle server response here
		nick = s
	}
	for {
		fmt.Print("Enter your username:\n> ")
		s := getInput()
		if strings.ToUpper(s) == "HELP" {
			continue
		}
		if s == "" {
			user = irc.DEFAULT_USER
		} else {
			user = s
		}
		break
	}
	for realname == "" {
		fmt.Print("Enter your real name:\n> ")
		s := getInput()
		if strings.ToUpper(s) == "HELP" {
			continue
		}
		realname = s
	}
	sendMsg("USER", user, 0, "*", ":"+realname)
}

func handleResponse() bool {
	buf := make([]byte, irc.BUFFER_SIZE)

	if read, _ := conn.Read(buf); read > 0 {
		switch string(buf[:read]) {
		case irc.RPL_NOP:
			return true
		case irc.ERR_NEEDMOREPARAMS:
			fmt.Println("Not enough parameters.")
			return false
		case irc.ERR_ALREADYREGISTRED:
			fmt.Println("Unauthorized command (already registered).")
			return false
		case irc.ERR_NOTREGISTERED:
			fmt.Println("You have not registered.")
			return false
		case irc.ERR_NONICKNAMEGIVEN:
			fmt.Println("You must specify your nickname.")
			return false
		case irc.ERR_GENERAL:
			fmt.Println("General error. Do you need HELP?")
			return false
		default:
			fmt.Printf("Response string: %s\n", buf)
		}
	}
	return true
}

func getInput() string {
	buf, err := reader.ReadBytes('\n')
	if err != nil {
		panic(err.Error())
	}
	s := string(buf)
	s = strings.TrimSuffix(s, "\r\n")
	s = strings.TrimSuffix(s, "\n")
	sUpper := strings.ToUpper(s)
	if sUpper == "QUIT" {
		panic(sUpper)
	}
	return s
}

// Writes a message in IRC format to the server.
// Command must be a string, and any following parameters may be of any type.
// If a parameter contains a space, it will be separated into two parameters by
// the server. The space(s) in a parameter may be preserved if it is the last
// parameter by prepending a colon (":") to the parameter argument.
func sendMsg(command string, a ...interface{}) {
	var buf []byte
	buf = append(buf, []byte(command)...)
	for _, val := range a {
		s := fmt.Sprint(" ", val)
		buf = append(buf, []byte(s)...)
	}
	buf = append(buf, []byte("\r\n")...)

	fmt.Print(string(buf))

	if len(buf) > irc.BUFFER_SIZE {
		panic("Message too large")
	}
	_, err := conn.Write(buf)
	if err != nil {
		panic(err.Error())
	}
}

// Program entry point
func main() {
	var err error
	printWelcome()

	// Attempt to connect to the server address and port via TCP
	conn, err = net.Dial(irc.PROTOCOL, irc.HOST_ADDRESS)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Create a byte reader for stdin
	reader = bufio.NewReaderSize(os.Stdin, irc.BUFFER_SIZE)
	register()

	// Read a line of text, and write it to the server
	for {
		fmt.Printf("> ")
		buf, err := reader.ReadBytes('\n')
		if err != nil {
			log.Fatalln(err)
		}
		s := strings.ToUpper(string(buf))
		if s == "QUIT\r\n" || s == "QUIT\n" {
			_, _ = conn.Write(buf)
			nRead, _ := conn.Read(buf)
			fmt.Printf("%s\n", buf[:nRead])
			conn.Close()
			return
		} else if s == "HELP\r\n" || s == "HELP\n" {
			printHelp()
			continue
		}
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalln(err)
		}
		go handleResponse()
	}
}
