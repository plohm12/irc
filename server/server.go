//TODO remove any leftover records when server terminates?
//TODO convert string returns to error returns
//TODO implement goroutine channels

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"irc"
	"irc/parser"
	"log"
	"net"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Contains unique ID, network connection, and receiving channel
type Session struct {
	id      int64
	conn    net.Conn
	receive chan string
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
		s.receive <- irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}

	// Query database for password
	err := db.QueryRow("SELECT password FROM users WHERE id=?", s.id).Scan(&password)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	}

	if password != "" {
		s.receive <- irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}
	_, err = db.Exec("UPDATE users SET password=? WHERE id=?", msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles NICK commands by updating the session record's nickname field.
// Returns an empty string on success or the appropriate error reply.
func (s *Session) handleNick(msg *parser.Message) {
	var password string
	var nickname string

	err := db.QueryRow("SELECT password,nickname FROM users WHERE id=?", s.id).Scan(&password, &nickname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	}

	if password == "" {
		s.receive <- irc.ERR_NOTREGISTERED + irc.CRLF
		return
	}
	if msg.Params.Num < 1 {
		s.receive <- irc.ERR_NONICKNAMEGIVEN + irc.CRLF
		return
	}
	//TODO check that nick fits spec
	//TODO check for collisions
	_, err = db.Exec("UPDATE users SET nickname=? WHERE id=?", msg.Params.Others[0], s.id)
	if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
	}
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func (s *Session) handleUser(msg *parser.Message) {
	var username string
	var realname string
	var mode int

	err := db.QueryRow("SELECT username,realname FROM users WHERE id=?", s.id).Scan(&username, &realname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	} else if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	}

	if msg.Params.Num < 4 {
		s.receive <- irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	if username != "" {
		s.receive <- irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	//TODO probably check that each field is safe
	mode, err = strconv.Atoi(msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}
	_, err = db.Exec("UPDATE users SET username=?,mode=?,realname=? WHERE id=?", msg.Params.Others[0], mode, msg.Params.Others[3], s.id)
	s.receive <- irc.RPL_WELCOME + irc.CRLF
}

// Handles QUIT commands by removing session record from database.
func (s *Session) handleQuit(msg *parser.Message) {
	_, err := db.Exec("DELETE FROM users WHERE id=?", s.id)
	if err != nil {
		panic(err)
	}
	s.receive <- irc.ERR_CONNCLOSED + irc.CRLF
}

func (s *Session) handlePrivMsg(msg *parser.Message) {
	var targetid int64
	var buf []byte
	err := db.QueryRow("SELECT id FROM users WHERE nickname=?", msg.Params.Others[0]).Scan(&targetid)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		s.receive <- errors.New(irc.ERR_NOSUCHNICK + " " + msg.Params.Others[0] + irc.CRLF).Error()
		return
	} else if err != nil {
		log.Println(err)
		s.receive <- irc.ERR_GENERAL + irc.CRLF
		return
	}

	if msg.Params.Num < 2 {
		s.receive <- irc.ERR_NOTEXTTOSEND + irc.CRLF
		return
	}
	targetSession, ok := sessions[targetid]
	if !ok {
		s.receive <- irc.ERR_NORECIPIENT + irc.CRLF
		return
	}
	buf = append(buf, []byte(msg.Command)...)
	for _, p := range msg.Params.Others {
		buf = append(buf, []byte(" "+p)...)
	}
	buf = append(buf, []byte(irc.CRLF)...)
	fmt.Print(string(buf)) // debug
	_, _ = targetSession.conn.Write(buf)
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
	}
}

// Handles session termination. Recovers from panicking within serve(). Deletes
// session record from database before closing connection.
func (s *Session) terminate() {
	defer s.conn.Close()
	fmt.Println("Terminating session", s.id)
	if err := recover(); err != nil {
		fmt.Println("Recovered:", err)
		//_, _ = conn.Write([]byte(fmt.Sprintf("%v", err)))
	}
	delete(sessions, s.id)
	_, err := db.Exec("DELETE FROM users WHERE id=?", s.id)
	if err != nil {
		panic(err)
	}
}

// Given a connection, read and print messages to console
//TODO after parsing message, check state info to determine if connection should stay open
func serve(conn net.Conn) {
	p := parser.NewParser(conn)

	// Create database record
	dbResult, err := db.Exec("INSERT INTO users () VALUES();")
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
		case out := <-s.receive:
			_, _ = conn.Write([]byte(out))
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
	// Access the database that stores state information
	var err error
	db, err = sql.Open(irc.DB_DRIVER, irc.DB_DATASOURCE)
	if err != nil {
		panic(err)
	}
	defer db.Close()

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
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go serve(conn)
	}
}
