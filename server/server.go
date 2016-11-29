//TODO: finish database stuff dummy!
//TODO: fix return statement args so they are less confusing
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

func handlePass(id int64, msg *parser.Message) (reply string) {
	var password string
	reply = ""
	if msg.Params.Num < 1 {
		reply = irc.ERR_NEEDMOREPARAMS
		return
	}

	// Query database for password
	err := db.QueryRow("SELECT password FROM users WHERE id=?", id).Scan(&password)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		reply = irc.ERR_GENERAL
		return
	} else if err != nil {
		log.Println(err)
		reply = irc.ERR_GENERAL
		return
	}

	if password != "" {
		reply = irc.ERR_ALREADYREGISTRED
		return
	}
	_, err = db.Exec("UPDATE users SET password=? WHERE id=?", msg.Params.Others[0], id)
	if err != nil {
		log.Println(err)
		reply = irc.ERR_GENERAL
	}
	return
}

func handleNick(id int64, msg *parser.Message) (reply string) {
	reply = ""
	var password string
	var nickname string

	err := db.QueryRow("SELECT password,nickname FROM users WHERE id=?", id).Scan(&password, &nickname)
	if err == sql.ErrNoRows {
		log.Println("No user with that ID.")
		reply = irc.ERR_GENERAL
		return
	} else if err != nil {
		log.Println(err)
		reply = irc.ERR_GENERAL
		return
	}

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
	_, err = db.Exec("UPDATE users SET nickname=? WHERE id=?", msg.Params.Others[0], id)
	if err != nil {
		log.Println(err)
		reply = irc.ERR_GENERAL
	}
	return
}

func handleUser(id int64, msg *parser.Message) (reply string) {
	reply = ""
	// if password == "" || nick == "" {
	// 	reply = irc.ERR_NOTREGISTERED
	// 	return
	// }
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

func handleQuit(id int64, msg *parser.Message) string {
	// Remove this client's record from the database
	_, err := db.Exec("DELETE FROM users WHERE id=?", id)
	if err != nil {
		log.Println(err)
	}
	return irc.ERR_CONNCLOSED
}

/* generic message handler */
func handleMessage(id int64, msg *parser.Message) (reply string) {
	reply = ""
	switch strings.ToUpper(msg.Command) {
	case "QUIT":
		reply = handleQuit(id, msg)
		//reply = irc.ERR_CONNCLOSED
	case "PASS":
		reply = handlePass(id, msg)
	case "NICK":
		reply = handleNick(id, msg)
	case "USER":
		reply = handleUser(id, msg)
	}
	return
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
				fmt.Println("A connection was closed.")
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
