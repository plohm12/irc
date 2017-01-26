//TODO add parser
//TODO enumerate help messages
//TODO add async message receiving

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"irc"
	"net"
	"os"
	"strings"
)

var (
	cbuf     bytes.Buffer
	conn     *net.Conn
	nick     string        = ""
	password string        = ""
	user     string        = ""
	mode     int           = 0
	realname string        = ""
	console  *bufio.Reader = bufio.NewReader(os.Stdin)
	input    chan string   = make(chan string, irc.CHAN_BUF_SIZE)
	incoming chan string   = make(chan string, irc.CHAN_BUF_SIZE)
)

// Program entry point
func main() {
	var message string
	var err error
	printWelcome()

	// Attempt to connect to the server address and port via TCP
	dialResult, err := net.Dial(irc.NETWORK, irc.HOST_ADDRESS)
	if err != nil {
		panic(err)
	}
	conn = &dialResult

	// Executes very last. This function recovers from panic() if necessary,
	// and closes the connection.
	defer func(c *net.Conn) {
		conn := *(c)
		if err := recover(); err != nil {
			if err == "/Q" || err == "/QUIT" {
				sendMsg("QUIT")
			} else {
				irc.Debug("Recovered:", err)
			}
		}
		conn.Close()
	}(conn)

	go receiveMessages()

	register()

	for {
		select {
		case message = <-incoming:
			fmt.Print(message)
		case message = <-input:
			command, params := getCommand(message)
			//if command != "" {
			handle(command, params)
			//}
		}
	}
}

func getInput() {
	var r rune
	var err error
	fmt.Print("> ")
	for {
		r, _, err = console.ReadRune()
		if err != nil {
			panic(err)
		}
		_, err = cbuf.WriteRune(r)
		if err != nil {
			panic(err)
		}
		if r == '\n' {
			input <- cbuf.String()
			cbuf.Reset()
			return
		}
	}
}

// Writes a message in IRC format to the server. Command must already be
// determined. If a parameter contains a space, it will be separated into two
// parameters by the server. The space(s) in a parameter may be preserved if it
// is the last parameter by prepending a colon (":") to the parameter argument.
// Returns nil on success, error otherwise.
func sendMsg(command string, args ...interface{}) error {
	var buf []byte
	buf = append(buf, []byte(command)...)
	for _, val := range args {
		s := fmt.Sprint(" ", val)
		buf = append(buf, []byte(s)...)
	}
	buf = append(buf, []byte("\r\n")...)

	//fmt.Print(string(buf)) // debug

	if len(buf) > irc.BUFFER_SIZE {
		return fmt.Errorf("Message length %v too big: %s", len(buf), string(buf))
	}
	conn := *conn
	_, err := conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

// Asyncrhonously waits for incoming messages from the server and pushes them to
// the channel.
func receiveMessages() {
	buf := make([]byte, irc.BUFFER_SIZE)
	conn := *conn
	for {
		numRead, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		if numRead > 0 {
			incoming <- string(buf[:numRead])
		}
	}
}

// Prints initial usage information.
func printWelcome() {
	fmt.Println("Welcome to the Internet Relay Chat!")
	fmt.Println("For HELP, type \"/help\" at any time.")
	fmt.Println(`To terminate your session, type "/quit" at any time.`)
}

// Prints a generic help message. Eventually will provide customized help for
// each command.
func printHelp(topic string) {
	switch topic {
	default:
		fmt.Println("Here is a list of available commands:")
		fmt.Println("  /help\t- display this help")
		fmt.Println("  /join <channel>\t- join a channel or make a new one")
		fmt.Println("  /part <channel>\t- leave a channel")
		fmt.Println("  /pm <target> <message>\t- send <message> to <target>")
		fmt.Println("  /q\t- same as /quit")
		fmt.Println("  /quit\t- terminate your session")
	}
}

// Called when user tries a command that does not exist.
func unrecognizedCommand(command string) {
	fmt.Println("Unrecognized command:", strings.ToUpper(command))
	fmt.Println(`Do you need "/help"?`)
}

// Attempt to parse a command from the input, which must be escaped by "/".
// Currently, commands are the only valid input.
func getCommand(msg string) (command string, trailing string) {
	var cmd []byte
	var cmdlen int
	if len(msg) == 0 {
		return "", ""
	}
	if msg[0] == '/' {
		cmdlen++
		for _, v := range msg[1:] {
			cmdlen++
			if v == ' ' {
				break
			}
			cmd = append(cmd, byte(v))
		}
	}
	command = string(cmd)
	trailing = string(msg[cmdlen:])
	return
}

// Examine the command to determine a course of action.
func handle(command string, params string) error {
	switch strings.ToUpper(command) {
	case "HELP":
		printHelp(params)
	case "PM":
		return handlePrivMsg(params)
	case "JOIN":
		return sendMsg("JOIN", params)
	case "PART":
		return sendMsg("PART", params)
	default:
		//unrecognizedCommand(command)
		return sendMsg("", params)
	}
	return nil
}

// Prepares a private message and calls sendMsg() to send it to the server.
func handlePrivMsg(params string) error {
	var target []byte
	var targetlen int

	for _, v := range params {
		targetlen++
		if v == ' ' {
			break
		}
		target = append(target, byte(v))
	}
	return sendMsg("PRIVMSG", string(target), ":"+params[targetlen:])
}

// Prompts the user for a session password, nickname, username, and real name.
// Sends PASS, NICK, & USER messages to server to establish a session.
func register() {
	// Loop until password is set
	for password == "" {
		fmt.Print("Create a session password:")
		go getInput()
		s := <-input
		if strings.ToUpper(s) == "/HELP\r\n" {
			continue
		}
		sendMsg("PASS", s)
		//		if handleResponse() {
		//			fmt.Println("Password success!")
		//			password = s
		//		}
		password = s
	}
	// Loop until nickname is set
	for nick == "" {
		fmt.Print("Enter your nickname:")
		go getInput()
		s := <-input
		if strings.ToUpper(s) == "/HELP\r\n" {
			continue
		}
		sendMsg("NICK", s)
		// handle server response here
		nick = s
	}
	fmt.Print("Enter your username (optional):")
	go getInput()
	s := <-input
	if s == "" {
		user = irc.DEFAULT_USER
	} else {
		user = s
	}
	// Loop until realname is set
	for realname == "" {
		fmt.Print("Enter your real name:")
		go getInput()
		s := <-input
		if strings.ToUpper(s) == "/HELP\r\n" {
			continue
		}
		realname = s
	}
	sendMsg("USER", user, 0, "*", ":"+realname)
}

// Currently not used.
//func handleResponse() bool {
//	buf := make([]byte, irc.BUFFER_SIZE)
//	conn := *conn

//	if read, _ := conn.Read(buf); read > 0 {
//		switch string(buf[:read]) {
//		case irc.RPL_NOP:
//			return true
//		case irc.ERR_NEEDMOREPARAMS:
//			fmt.Println("Not enough parameters.")
//			return false
//		case irc.ERR_ALREADYREGISTRED:
//			fmt.Println("Unauthorized command (already registered).")
//			return false
//		case irc.ERR_NOTREGISTERED:
//			fmt.Println("You have not registered.")
//			return false
//		case irc.ERR_NONICKNAMEGIVEN:
//			fmt.Println("You must specify your nickname.")
//			return false
//		case irc.ERR_GENERAL:
//			fmt.Println("General error. Do you need /HELP?")
//			return false
//		default:
//			fmt.Printf("Response string: %s\n", buf)
//		}
//	}
//	return true
//}
