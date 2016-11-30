//TODO: remove any leftover records when server terminates?

package main

import (
	"database/sql"
	"fmt"
	"irc"
	"irc/parser"
	"log"
	"net"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
)

// Handles PASS commands by updating the session record's password field.
// Returns an empty string on success or the appropriate error reply.
func handlePass(id int64, msg *parser.Message) string {
	var password string
	if msg.Params.Num < 1 {
		return irc.ERR_NEEDMOREPARAMS
	}

	// Query database for password
	err := db.QueryRow("SELECT password FROM users WHERE id=?", id).Scan(&password)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		return irc.ERR_GENERAL
	} else if err != nil {
		log.Println(err)
		return irc.ERR_GENERAL
	}

	if password != "" {
		return irc.ERR_ALREADYREGISTRED
	}
	_, err = db.Exec("UPDATE users SET password=? WHERE id=?", msg.Params.Others[0], id)
	if err != nil {
		log.Println(err)
		return irc.ERR_GENERAL
	}
	return ""
}

// Handles NICK commands by updating the session record's nickname field.
// Returns an empty string on success or the appropriate error reply.
func handleNick(id int64, msg *parser.Message) string {
	var password string
	var nickname string

	err := db.QueryRow("SELECT password,nickname FROM users WHERE id=?", id).Scan(&password, &nickname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		return irc.ERR_GENERAL
	} else if err != nil {
		log.Println(err)
		return irc.ERR_GENERAL
	}

	if password == "" {
		return irc.ERR_NOTREGISTERED
	}
	if msg.Params.Num < 1 {
		return irc.ERR_NONICKNAMEGIVEN
	}
	//TODO: check that nick fits spec
	//TODO: check for collisions
	_, err = db.Exec("UPDATE users SET nickname=? WHERE id=?", msg.Params.Others[0], id)
	if err != nil {
		log.Println(err)
		return irc.ERR_GENERAL
	}
	return ""
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func handleUser(id int64, msg *parser.Message) string {
	var username string
	var realname string
	var mode int

	err := db.QueryRow("SELECT username,realname FROM users WHERE id=?", id).Scan(&username, &realname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		return irc.ERR_GENERAL
	} else if err != nil {
		log.Println(err)
		return irc.ERR_GENERAL
	}

	if msg.Params.Num < 4 {
		return irc.ERR_NEEDMOREPARAMS
	}
	if username != "" {
		return irc.ERR_ALREADYREGISTRED
	}

	//TODO: probably check that each field is safe
	mode, err = strconv.Atoi(msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}
	_, err = db.Exec("UPDATE users SET username=?,mode=?,realname=? WHERE id=?", msg.Params.Others[0], mode, msg.Params.Others[3], id)
	return irc.RPL_WELCOME
}

// Handles QUIT commands by removing session record from database.
func handleQuit(id int64, msg *parser.Message) string {
	_, err := db.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		panic(err)
	}
	return irc.ERR_CONNCLOSED
}

/* generic message handler */
func handleMessage(id int64, msg *parser.Message) string {
	switch strings.ToUpper(msg.Command) {
	case "QUIT":
		return handleQuit(id, msg)
		//return irc.ERR_CONNCLOSED
	case "PASS":
		return handlePass(id, msg)
	case "NICK":
		return handleNick(id, msg)
	case "USER":
		return handleUser(id, msg)
	}
	return ""
}

// Handles session termination. Recovers from panicking within serve(). Deletes
// session record from database before closing connection.
func closeConnection(conn net.Conn, id int64) {
	defer conn.Close()
	fmt.Println("Terminating session", id)
	if err := recover(); err != nil {
		fmt.Println("Recovered and sending:", err)
		_, _ = conn.Write([]byte(fmt.Sprintf("%v", err)))
	}
	_, err := db.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		panic(err)
	}
}

// Given a connection, read and print messages to console
//TODO: after parsing message, check state info to determine if connection should stay open
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
	fmt.Println("Created session", id)
	defer closeConnection(conn, id)

	for {
		if msg, err := p.Parse(); err != nil {
			panic(err)
		} else {
			parser.Print(msg)
			reply := handleMessage(id, msg)
			_, _ = conn.Write([]byte(reply))
			if reply == irc.ERR_CONNCLOSED {
				return
			}
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
