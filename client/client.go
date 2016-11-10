package main

import(
	"bufio"
	//"bytes"
	"fmt"
	"irc"
	"log"
	"net"
	"os"
	"strings"
	//"time"
)

var(
	nick	string	=	"";
)

func printWelcome() {
	fmt.Println("Welcome to the Internet Relay Chat!")
	fmt.Println("To register with a server, you must enter three messages: PASS, NICK, and USER.")
	fmt.Println("For HELP, type \"help\".")
	fmt.Println("To terminate your session, type \"quit\".")
}

func printHelp() {
	fmt.Println("Enter: PASS [password]")
}

func handleResponse(conn net.Conn) {
	buf := make([]byte, 512)
	//conn.SetReadDeadline(time.Now().Add(time.Second))

	if read, _ := conn.Read(buf); read > 0 {
		switch string(buf[:read]) {
		case irc.ERR_NEEDMOREPARAMS:
			fmt.Println("Need more params.")
		case irc.ERR_ALREADYREGISTRED:
			fmt.Println("You've already registered a password.")
		case irc.ERR_GENERAL:
			fmt.Println("General error. Do you need HELP?")
		default:
			fmt.Printf("Response string: %s\n", buf)
		}
	}
}

// Program entry point
func main() {
	printWelcome()

	// Attempt to connect to the server address and port via TCP
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Create a byte reader for stdin
	reader := bufio.NewReaderSize(os.Stdin, 512)

	// Read a line of text, and write it to the server
	for {
		fmt.Printf("> ")
		buf, err := reader.ReadBytes(10) // 10 == '\n'
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
		go handleResponse(conn)
	}
}
