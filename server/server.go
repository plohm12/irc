//TODO: finish database stuff dummy!
//TODO: remove any leftover records when server terminates

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
	db       *sql.DB
	nick     string = ""
	username string = ""
	mode     int    = 0
	realname string = ""
)

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

func handleUser(id int64, msg *parser.Message) string {
	// if password == "" || nick == "" {
	// 	return irc.ERR_NOTREGISTERED
	// }
	if msg.Params.Num < 4 {
		return irc.ERR_NEEDMOREPARAMS
	}
	if username != "" {
		return irc.ERR_ALREADYREGISTRED
	}

	//TODO: probably check that each field is safe
	username = msg.Params.Others[0]
	if m, err := strconv.Atoi(msg.Params.Others[1]); err != nil {
		log.Println(err)
		username = ""
		mode = 0
		return irc.ERR_GENERAL
	} else {
		mode = m
	}
	// discard 3rd param; it is unused
	realname = msg.Params.Others[3]
	return irc.RPL_WELCOME
}

func handleQuit(id int64, msg *parser.Message) string {
	// Remove this client's record from the database
	_, err := db.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		log.Println(err)
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

func closeConnection(conn net.Conn, id int64) {
	fmt.Println("Serve() ending for userid", id)
	if err := recover(); err != nil {
		fmt.Println("Recovered:", err)
		_, _ = conn.Write([]byte(fmt.Sprintf("%v", err)))
	}
	_, err := db.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		log.Println(err)
	}
	conn.Close()
}

// Given a connection, read and print messages to console
//TODO: after parsing message, check state info to determine if connection should stay open
func serve(conn net.Conn) {
	p := parser.NewParser(conn)
	fmt.Println("A connection was opened.")

	// Create database record
	dbResult, err := db.Exec("INSERT INTO users () VALUES();")
	if err != nil {
		log.Println(err)
	}
	id, err := dbResult.LastInsertId()
	if err != nil {
		log.Println(err)
	}
	defer closeConnection(conn, id)

	for {
		if msg, err := p.Parse(); err != nil {
			log.Println(err)
			_, _ = conn.Write([]byte(irc.ERR_GENERAL))
			conn.Close()
			return
		} else {
			parser.Print(msg)
			reply := handleMessage(id, msg)
			_, _ = conn.Write([]byte(reply))
			if reply == irc.ERR_CONNCLOSED {
				conn.Close()
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
		panic(err.Error())
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// Listen for TCP connections on this address and port
	ln, err := net.Listen(irc.NETWORK, irc.HOST_ADDRESS)
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
