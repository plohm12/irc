//TODO remove unnecessary code
//TODO migrate database create/delete

package main

import (
	"fmt"
	"irc"
	"irc/database"
	"irc/parser"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

// Contains unique client ID, network connection, and a received message
type Session struct {
	id   database.Id
	conn *net.Conn
	msg  *parser.Message
}

var (
	// Maps session IDs to Session struct
	sessions map[database.Id]*Session = make(map[database.Id]*Session)
	// buffered channel for receiving communications
	received chan *Session = make(chan *Session, 10)
)

// Program entry point
func main() {
	killswitch := makeKillSwitch()
	database.Create()
	go listenAndServe()
	go handleMessages()
	fmt.Println("Welcome to the IRC server!")

	// Block until keyboard interrupt (e.g. ctrl-C).
	// Everything following is clean-up before exit.
	// DOES NOT WORK WITH MinGW!!
	<-killswitch
	fmt.Println("ABORTING PROGRAM...")
	database.Destroy()
	fmt.Println("Goodbye!")
}

// Handles PASS commands by updating the session record's password field.
// Returns an empty string on success or the appropriate error reply.
func (s *Session) handlePass() {
	if s.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}

	// Query database for password
	password, ok := s.id.GetPassword()
	if !ok {
		// NO USER WITH THAT ID. DO SOMETHING
		return
	}

	if password != "" {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	s.id.SetPassword(s.msg.Params.Others[0])
}

// Handles NICK commands by updating the session record's nickname field.
func (s *Session) handleNick() {
	nickname, ok := s.id.GetNickname()
	if !ok {
		// NO USER WITH THAT ID. DO SOMETHING
		return
	}
	if nickname != "" {
		fmt.Println(nickname)
	}

	if s.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NONICKNAMEGIVEN + irc.CRLF
		return
	}
	//TODO check that nick fits spec
	//TODO check for collisions
	s.id.SetNickname(s.msg.Params.Others[0])
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func (s *Session) handleUser() {
	var mode int

	username, realname, ok := s.id.GetUsernameRealname()
	if !ok {
		// NO USER WITH THAT ID
		return
	}

	if s.msg.Params.Num < 4 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	if username != "" {
		fmt.Println(realname)
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	//TODO probably check that each field is safe
	mode, err := strconv.Atoi(s.msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}

	s.id.SetUsernameModeRealname(s.msg.Params.Others[0], s.msg.Params.Others[3], mode)
}

// Handles QUIT commands by removing session record from database.
func (s *Session) handleQuit() {
	if err := database.DeleteUser(s.id); err != nil {
		panic(err)
	}
}

// Sends a message to target, which should be the first parameter. Target is
// either a nick, user, or channel.
func (s *Session) handlePrivMsg() {
	var buf []byte

	nickname, username, ok := s.id.GetNicknameUsername()
	if !ok {
		//not ok
	}
	senderPrefix := ":" + nickname + "!" + username + "@" + irc.HOST_IP
	fmt.Print(senderPrefix)

	// make sure there is a target and message to send
	if s.msg.Params.Num < 2 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOTEXTTOSEND + " " + nickname + irc.CRLF
		return
	}

	//should this message be broadcast to a channel?
	if s.msg.Params.Others[0][0] == '#' {
		if _, ok := database.GetChannelCreator(s.msg.Params.Others[0]); !ok {
			//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CANNOTSENDTOCHAN + irc.CRLF
			return
		}
		users := database.GetChannelUsers(s.msg.Params.Others[0])
		for _, userid := range users {
			fmt.Print(userid)
			//			targetSession, ok := sessions[userid]
			//			if !ok {
			//				continue
			//			}
			//			targetSession.ch <- senderPrefix + " " + s.msg.Command + " " + s.msg.Params.Others[0] + " " + s.msg.Params.Others[1] + irc.CRLF
		}
	} else {
		// not a channel message, try to send PM to target user
		var targetid database.Id
		targetid, ok := database.GetIdByNickname(s.msg.Params.Others[0])
		if !ok {
			// something's not ok
		}

		targetSession, ok := sessions[targetid]
		if !ok {
			fmt.Print(targetSession)
			//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NORECIPIENT + irc.CRLF
			return
		}
		buf = append(buf, []byte(":"+nickname+"!"+username+"@"+irc.HOST_IP+" ")...)
		buf = append(buf, []byte(s.msg.Command)...)
		for _, p := range s.msg.Params.Others {
			buf = append(buf, []byte(" "+p)...)
		}
		buf = append(buf, []byte(irc.CRLF)...)
		fmt.Print(string(buf)) // debug
		//_, _ = targetSession.conn.Write(buf)
		//targetSession.ch <- string(buf)

	}
}

func (s *Session) handleJoin() {
	if s.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	_, err := database.JoinChannel(s.msg.Params.Others[0], s.id)
	if err != nil {
		//s.ch <- err.Error() // is this correct?
		return
	}
	//s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_TOPIC + " " + s.msg.Params.Others[0] + " :" + topic + irc.CRLF
}

func (s *Session) handlePart() {
	if s.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	err := database.PartChannel(s.msg.Params.Others[0], s.id)
	if err != nil {
		//s.ch <- err.Error()
	}

}

// generic message handler
func (s *Session) handle() {
	switch strings.ToUpper(s.msg.Command) {
	case "QUIT":
		s.handleQuit()
	case "PASS":
		s.handlePass()
	case "NICK":
		s.handleNick()
	case "USER":
		s.handleUser()
	case "PRIVMSG":
		s.handlePrivMsg()
	case "JOIN":
		s.handleJoin()
	case "PART":
		s.handlePart()
	}
}

// Parse messages from a client and queue them for handling
func serve(conn *net.Conn, id database.Id) {
	defer func(id database.Id) {
		if r := recover(); r != nil {
			fmt.Println("serve(", id, ") error:", r)
		}
	}(id)

	p := parser.NewParser(*conn)
	for {
		msg, err := p.Parse()
		if err != nil {
			panic(err)
		}
		parser.Print(msg) // debug
		s := &Session{id, conn, msg}
		received <- s
	}
}

// When a message is received, dispatch a goroutine to handle it
func handleMessages() {
	for r := range received {
		go r.handle() //.reply()
	}
}

// Establish network connection, and accept incoming connections. Dispatch
// goroutines for new connections to separately serve them.
func listenAndServe() {
	listener, err := net.Listen(irc.NETWORK, irc.HOST_ADDRESS)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		id := database.CreateUser()
		fmt.Println("Created session", id)
		go serve(&conn, id)
	}
}

// Create a channel for intercepting keyboard interrupts, the purpose of
// which is to prevent unsafe program termination.
func makeKillSwitch() chan os.Signal {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGTERM)
	return interrupt
}
