//TODO convert string returns to error returns
//TODO implement goroutine channels

package main

import (
	"database/sql"
	"fmt"
	"irc"
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

// Contains unique ID, network connection, and receiving channel
type Session struct {
	id   int64
	conn net.Conn
	ch   chan string
}

var (
	// Global database identifier
	db *sql.DB
	// Maps session IDs to Session struct
	sessions map[int64]*Session = make(map[int64]*Session)
)

// Handles PASS commands by updating the session record's password field.
// Returns an empty string on success or the appropriate error reply.
func (s *Session) handlePass(msg *parser.Message) {
	var password string
	if msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}

	// Query database for password
	err := db.QueryRow("SELECT password FROM "+irc.TABLE_USERS+" WHERE id=?", s.id).Scan(&password)
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
	_, err = db.Exec("UPDATE users SET password=? WHERE id=?", msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles NICK commands by updating the session record's nickname field.
// Returns an empty string on success or the appropriate error reply.
func (s *Session) handleNick(msg *parser.Message) {
	var password string
	var nickname string

	err := db.QueryRow("SELECT password,nickname FROM "+irc.TABLE_USERS+" WHERE id=?", s.id).Scan(&password, &nickname)
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
	if msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NONICKNAMEGIVEN + irc.CRLF
		return
	}
	//TODO check that nick fits spec
	//TODO check for collisions
	_, err = db.Exec("UPDATE users SET nickname=? WHERE id=?", msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func (s *Session) handleUser(msg *parser.Message) {
	var username string
	var realname string
	var mode int

	err := db.QueryRow("SELECT username,realname FROM "+irc.TABLE_USERS+" WHERE id=?", s.id).Scan(&username, &realname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_GENERAL + irc.CRLF
		return
	}

	if msg.Params.Num < 4 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	if username != "" {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	//TODO probably check that each field is safe
	mode, err = strconv.Atoi(msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}
	_, err = db.Exec("UPDATE users SET username=?,mode=?,realname=? WHERE id=?", msg.Params.Others[0], mode, msg.Params.Others[3], s.id)
	s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_WELCOME + " " + username + irc.CRLF
}

// Handles QUIT commands by removing session record from database.
func (s *Session) handleQuit(msg *parser.Message) {
	_, err := db.Exec("DELETE FROM "+irc.TABLE_USERS+" WHERE id=?", s.id)
	if err != nil {
		panic(err)
	}
	s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CONNCLOSED + irc.CRLF
}

// Sends a message to target, which should be the first parameter. Target is
// either a nick, user, or channel.
func (s *Session) handlePrivMsg(msg *parser.Message) {
	var nickname string
	var username string
	var senderPrefix string
	var buf []byte

	_ = db.QueryRow("SELECT nickname,username FROM "+irc.TABLE_USERS+" WHERE id=?", s.id).Scan(&nickname, &username)
	senderPrefix = ":" + nickname + "!" + username + "@" + irc.HOST_IP

	// make sure there is a target and message to send
	if msg.Params.Num < 2 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOTEXTTOSEND + " " + nickname + irc.CRLF
		return
	}

	//should this message be broadcast to a channel?
	if msg.Params.Others[0][0] == '#' {
		var creatorid int64 // dummy value
		err := db.QueryRow("SELECT creator FROM "+irc.TABLE_CHANNELS+" WHERE channel_name=?", msg.Params.Others[0]).Scan(&creatorid)
		if err == sql.ErrNoRows {
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CANNOTSENDTOCHAN + irc.CRLF
			return
		}
		rows, err := db.Query("SELECT user_id FROM "+irc.TABLE_USER_CHANNEL+" WHERE channel_name=?", msg.Params.Others[0])
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
			targetSession.ch <- senderPrefix + " " + msg.Command + " " + msg.Params.Others[0] + " " + msg.Params.Others[1] + irc.CRLF
		}
	} else {
		// not a channel message, try to send PM to target user
		var targetid int64
		err := db.QueryRow("SELECT id FROM "+irc.TABLE_USERS+" WHERE nickname=?", msg.Params.Others[0]).Scan(&targetid)
		if err == sql.ErrNoRows {
			log.Println("No user with that ID.")
			s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOSUCHNICK + " " + msg.Params.Others[0] + irc.CRLF
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
		buf = append(buf, []byte(msg.Command)...)
		for _, p := range msg.Params.Others {
			buf = append(buf, []byte(" "+p)...)
		}
		buf = append(buf, []byte(irc.CRLF)...)
		fmt.Print(string(buf)) // debug
		//_, _ = targetSession.conn.Write(buf)
		targetSession.ch <- string(buf)

	}
}

func (s *Session) handleJoin(msg *parser.Message) {
	if msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	topic, err := irc.JoinChannel(db, msg.Params.Others[0], s.id)
	if err != nil {
		s.ch <- err.Error() // is this correct?
		return
	}
	s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_TOPIC + " " + msg.Params.Others[0] + " :" + topic + irc.CRLF
}

func (s *Session) handlePart(msg *parser.Message) {
	if msg.Params.Num < 1 {
		s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	err := irc.PartChannel(db, msg.Params.Others[0], s.id)
	if err != nil {
		s.ch <- err.Error()
	}

}

/* generic message handler */
func (s *Session) handle(msg *parser.Message) {
	switch strings.ToUpper(msg.Command) {
	case "QUIT":
		s.handleQuit(msg)
	case "PASS":
		s.handlePass(msg)
	case "NICK":
		s.handleNick(msg)
	case "USER":
		s.handleUser(msg)
	case "PRIVMSG":
		s.handlePrivMsg(msg)
	case "JOIN":
		s.handleJoin(msg)
	case "PART":
		s.handlePart(msg)
	}
}

// Handles session termination. Recovers from panicking within serve(). Deletes
// session record from database before closing connection.
//TODO safely part from all channels
func (s *Session) terminate() {
	defer s.conn.Close()
	fmt.Println("Terminating session", s.id)
	if err := recover(); err != nil {
		fmt.Println("Recovered:", err)
		//_, _ = conn.Write([]byte(fmt.Sprintf("%v", err)))
	}
	delete(sessions, s.id)
	_, err := db.Exec("DELETE FROM "+irc.TABLE_USERS+" WHERE id=?", s.id)
	if err != nil {
		panic(err)
	}
}

// Given a connection, read and print messages to console
//TODO after parsing message, check state info to determine if connection should stay open
func serve(conn net.Conn) {
	p := parser.NewParser(conn)
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovered:", err)
		}
	}()

	// Create database record
	dbResult, err := db.Exec("INSERT INTO " + irc.TABLE_USERS + " () VALUES();")
	if err != nil {
		panic(err)
	}
	id, err := dbResult.LastInsertId()
	if err != nil {
		panic(err)
	}

	s := &Session{id, conn, make(chan string, irc.CHAN_BUF_SIZE)}
	sessions[id] = s
	fmt.Println("Created session", id)
	defer s.terminate()

	// Repeatedly handle messages
	for {
		select {
		case outgoing := <-s.ch:
			_, _ = conn.Write([]byte(outgoing))
		default:
			msg, err := p.Parse()
			if err != nil {
				panic(err)
			}
			parser.Print(msg) // debug
			s.handle(msg)
		}
	}
}

// Program entry point
func main() {
	var err error

	// Capture these signals and send them to the channel
	interruptChannel := make(chan os.Signal)
	signal.Notify(interruptChannel,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGTERM)

	db = irc.CreateDB()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()

	// Test the connection
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	// Listen for TCP connections on this address and port
	ln, err := net.Listen(irc.NETWORK, irc.HOST_ADDRESS)
	if err != nil {
		panic(err)
	}

	// Accept and serve each connection in a new goroutine
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			go serve(conn)
		}
	}()

	// Block until signal (e.g. ctrl-C).
	// Everything following is clean-up before exit.
	// DOES NOT WORK WITH MinGW!!
	<-interruptChannel
	fmt.Println("ABORTING PROGRAM...")
	irc.DestroyDB()
	fmt.Println("Goodbye!")
}
