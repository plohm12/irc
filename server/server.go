//TODO no zombie goroutines
//TODO safe exit from Parse()

package main

import (
	"fmt"
	"irc"
	"irc/database"
	"irc/message"
	"irc/message/parser"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	id   database.Id
	conn *net.Conn
}

// Contains unique client ID, network connection, and a received message
type Received struct {
	client *Client
	msg    *message.Message
}

type Reply struct {
	client *Client
	msg    *[]byte
}

var (
	sessions map[database.Id]*Client = make(map[database.Id]*Client)
	incoming chan *Received          = make(chan *Received, irc.CHAN_BUF_SIZE)
	outgoing chan *Reply             = make(chan *Reply, irc.CHAN_BUF_SIZE)
)

// Program entry point
func main() {
	killswitch := makeKillSwitch()
	database.Create()
	go listenAndServe()
	go receive()
	go reply()
	fmt.Println("Welcome to the IRC server!")

	// Block until keyboard interrupt (e.g. ctrl-C).
	// Everything following is clean-up before exit.
	// DOES NOT WORK WITH MinGW!!
	<-killswitch
	fmt.Println("ABORTING PROGRAM...")
	close(incoming)
	close(outgoing)
	database.Destroy()
	fmt.Println("Goodbye!")
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
		fmt.Println("Created client", id)
		go serve(&Client{id, &conn})
	}
}

// Parse messages from a client and queue them for handling
func serve(client *Client) {
	defer func(id database.Id) {
		if r := recover(); r != nil {
			fmt.Println("serve(", id, ") error:", r)
		}
	}(client.id)

	//TODO have Parse block on channel
	//TODO add exit channel to prevent zombie goroutines
	p := parser.NewParser(*client.conn)
	for {
		msg, err := p.Parse()
		if err != nil {
			panic(err)
		}
		message.Print(msg) // debug
		s := &Received{client, msg}
		incoming <- s
	}
}

// When a message is received, dispatch a goroutine to handle it
func receive() {
	for in := range incoming {
		go in.handle()
	}
}

func reply() {
	for out := range outgoing {
		go func(r *Reply) {
			conn := *r.client.conn
			msg := *r.msg
			_, err := conn.Write(msg)
			if err != nil {
				panic(err)
			}
		}(out)
	}
}

// generic message handler
func (r *Received) handle() {
	switch strings.ToUpper(r.msg.Command) {
	case "QUIT":
		r.handleQuit()
	case "PASS":
		r.handlePass()
	case "NICK":
		r.handleNick()
	case "USER":
		r.handleUser()
	case "PRIVMSG":
		r.handlePrivMsg()
	case "JOIN":
		r.handleJoin()
	case "PART":
		r.handlePart()
	}
}

// Handles PASS commands by updating the session record's password field.
// Returns an empty string on success or the appropriate error reply.
func (r *Received) handlePass() {
	//var reply []byte
	if r.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}

	// Query database for password
	password, ok := r.client.id.GetPassword()
	if !ok {
		// NO USER WITH THAT ID. DO SOMETHING
		return
	}

	if password != "" {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	r.client.id.SetPassword(r.msg.Params.Others[0])
}

// Handles NICK commands by updating the session record's nickname field.
func (r *Received) handleNick() {
	nickname, ok := r.client.id.GetNickname()
	if !ok {
		// NO USER WITH THAT ID. DO SOMETHING
		return
	}
	if nickname != "" {
		fmt.Println(nickname)
	}

	if r.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NONICKNAMEGIVEN + irc.CRLF
		return
	}
	//TODO check that nick fits spec
	//TODO check for collisions
	r.client.id.SetNickname(r.msg.Params.Others[0])
}

// Handles USER commands by updating session record with username and mode
// (along with user's real name). Returns a welcome reply on success or the
// appropriate error reply.
func (r *Received) handleUser() {
	var mode int

	username, realname, ok := r.client.id.GetUsernameRealname()
	if !ok {
		// NO USER WITH THAT ID
		return
	}

	if r.msg.Params.Num < 4 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	if username != "" {
		fmt.Println(realname)
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_ALREADYREGISTRED + irc.CRLF
		return
	}

	//TODO probably check that each field is safe
	mode, err := strconv.Atoi(r.msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}

	r.client.id.SetUsernameModeRealname(r.msg.Params.Others[0], r.msg.Params.Others[3], mode)
}

// Handles QUIT commands by removing session record from database.
func (r *Received) handleQuit() {
	database.DeleteUser(r.client.id)
	//TODO probably close conn or signal channel exit too
}

// Sends a message to target, which should be the first parameter. Target is
// either a nick, user, or channel.
func (r *Received) handlePrivMsg() {
	var buf []byte

	nickname, username, ok := r.client.id.GetNicknameUsername()
	if !ok {
		//not ok
	}
	senderPrefix := ":" + nickname + "!" + username + "@" + irc.HOST_IP
	fmt.Print(senderPrefix)

	// make sure there is a target and message to send
	if r.msg.Params.Num < 2 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NOTEXTTOSEND + " " + nickname + irc.CRLF
		return
	}

	//should this message be broadcast to a channel?
	if r.msg.Params.Others[0][0] == '#' {
		if _, ok := database.GetChannelCreator(r.msg.Params.Others[0]); !ok {
			//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_CANNOTSENDTOCHAN + irc.CRLF
			return
		}
		users := database.GetChannelUsers(r.msg.Params.Others[0])
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
		targetid, ok := database.GetIdByNickname(r.msg.Params.Others[0])
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
		buf = append(buf, []byte(r.msg.Command)...)
		for _, p := range r.msg.Params.Others {
			buf = append(buf, []byte(" "+p)...)
		}
		buf = append(buf, []byte(irc.CRLF)...)
		fmt.Print(string(buf)) // debug
		//_, _ = targetSession.conn.Write(buf)
		//targetSession.ch <- string(buf)

	}
}

func (r *Received) handleJoin() {
	if r.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	_ = database.JoinChannel(r.msg.Params.Others[0], r.client.id)
	//s.ch <- irc.SERVER_PREFIX + " " + irc.RPL_TOPIC + " " + s.msg.Params.Others[0] + " :" + topic + irc.CRLF
}

func (r *Received) handlePart() {
	if r.msg.Params.Num < 1 {
		//s.ch <- irc.SERVER_PREFIX + " " + irc.ERR_NEEDMOREPARAMS + irc.CRLF
		return
	}
	database.PartChannel(r.msg.Params.Others[0], r.client.id)
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
