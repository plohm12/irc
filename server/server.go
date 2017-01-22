//TODO remove unnecessary code
//TODO migrate database create/delete

package main

import (
	"database/sql"
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
	id   int64
	conn *net.Conn
	msg  *parser.Message
	ch   chan string
}

var (
	// Maps session IDs to Session struct
	sessions map[int64]*Session = make(map[int64]*Session)
	// buffered channel for receiving communications
	received chan *Session = make(chan *Session, 10)
)

// Program entry point
func main() {
	killswitch := makeKillSwitch()
	database.Create()
	go listenAndServe()
	go handleMessages()

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
	var password string
	if s.msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}

	// Query database for password
	err := db.QueryRow("SELECT password FROM "+database.TABLE_USERS+" WHERE id=?", s.id).Scan(&password)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	}

	if password != "" {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}
	_, err = db.Exec("UPDATE users SET password=? WHERE id=?", s.msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles NICK commands by updating the session record's nickname field.
// Returns an empty string on success or the appropriate error reply.
func (s *Session) handleNick() {
	var password string
	var nickname string

	err := db.QueryRow("SELECT password,nickname FROM "+database.TABLE_USERS+" WHERE id=?", s.id).Scan(&password, &nickname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	}

	if password == "" {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOTREGISTERED + irc.CRLF
		return
	}
	if s.msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NONICKNAMEGIVEN + irc.CRLF
		return
	}
	//TODO check that nick fits spec
	//TODO check for collisions
	_, err = db.Exec("UPDATE users SET nickname=? WHERE id=?", s.msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func (s *Session) handleUser() {
	var username string
	var realname string
	var mode int

	err := db.QueryRow("SELECT username,realname FROM "+database.TABLE_USERS+" WHERE id=?", s.id).Scan(&username, &realname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	}

	if s.msg.Params.Num < 4 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	if username != "" {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	//TODO probably check that each field is safe
	mode, err = strconv.Atoi(s.msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}
	_, err = db.Exec("UPDATE users SET username=?,mode=?,realname=? WHERE id=?", s.msg.Params.Others[0], mode, s.msg.Params.Others[3], s.id)
	s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_WELCOME + " " + username + irc.CRLF
}

// Handles QUIT commands by removing session record from database.
func (s *Session) handleQuit() {
	if err := database.DeleteUser(s.id); err != nil {
		panic(err)
	}
	s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CONNCLOSED + irc.CRLF
}

// Sends a message to target, which should be the first parameter. Target is
// either a nick, user, or channel.
func (s *Session) handlePrivMsg() {
	var nickname string
	var username string
	var senderPrefix string
	var buf []byte

	_ = db.QueryRow("SELECT nickname,username FROM "+database.TABLE_USERS+" WHERE id=?", s.id).Scan(&nickname, &username)
	senderPrefix = ":" + nickname + "!" + username + "@" + irc.HOST_IP

	// make sure there is a target and message to send
	if s.msg.Params.Num < 2 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOTEXTTOSEND + " " + nickname + irc.CRLF
		return
	}

	//should this message be broadcast to a channel?
	if s.msg.Params.Others[0][0] == '#' {
		var creatorid int64 // dummy value
		err := db.QueryRow("SELECT creator FROM "+database.TABLE_CHANNELS+" WHERE channel_name=?", s.msg.Params.Others[0]).Scan(&creatorid)
		if err == sql.ErrNoRows {
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CANNOTSENDTOCHAN + irc.CRLF
			return
		}
		rows, err := db.Query("SELECT user_id FROM "+database.TABLE_USER_CHANNEL+" WHERE channel_name=?", s.msg.Params.Others[0])
		if err != nil {
			//do something
		}
		defer rows.Close()
		for rows.Next() {
			var channeluserid int64
			rows.Scan(&channeluserid)
			targetSession, ok := sessions[channeluserid]
			if !ok {
				continue
			}
			targetSession.ch <- senderPrefix + " " + s.msg.Command + " " + s.msg.Params.Others[0] + " " + s.msg.Params.Others[1] + irc.CRLF
		}
	} else {
		// not a channel message, try to send PM to target user
		var targetid int64
		err := db.QueryRow("SELECT id FROM "+database.TABLE_USERS+" WHERE nickname=?", s.msg.Params.Others[0]).Scan(&targetid)
		if err == sql.ErrNoRows {
			log.Println("No user with that ID.")
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOSUCHNICK + " " + s.msg.Params.Others[0] + irc.CRLF
			return
		} else if err != nil {
			log.Println(err)
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
			return
		}

		targetSession, ok := sessions[targetid]
		if !ok {
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NORECIPIENT + irc.CRLF
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
		targetSession.ch <- string(buf)

	}
}

func (s *Session) handleJoin() {
	if s.msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	topic, err := database.JoinChannel(s.msg.Params.Others[0], s.id)
	if err != nil {
		s.ch <- err.Error() // is this correct?
		return
	}
	s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_TOPIC + " " + s.msg.Params.Others[0] + " :" + topic + irc.CRLF
}

func (s *Session) handlePart() {
	if s.msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	err := database.PartChannel(s.msg.Params.Others[0], s.id)
	if err != nil {
		s.ch <- err.Error()
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
func serve(conn *net.Conn, id int64) {
	defer func(id int64) {
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
		s := &Session{id, conn, msg, nil}
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
