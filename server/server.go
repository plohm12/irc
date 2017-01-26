//TODO no zombie goroutines
//TODO safe exit from Parse()

package main

import (
	"bytes"
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
		irc.Debug("Created client", id)
		client := &Client{id, &conn}
		sessions[id] = client
		go serve(client)
	}
}

// Parse messages from a client and queue them for handling
func serve(client *Client) {
	defer func(client Client) {
		if r := recover(); r != nil {
			irc.Debug("serve(", client.id, ") error:", r)
		}
		client.handleQuit()
	}(*client)

	//TODO have Parse block on channel
	//TODO add exit channel to prevent zombie goroutines
	p := parser.NewParser(*client.conn)
	for {
		msg, err := p.Parse()
		if err != nil {
			irc.Debug(err)
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
			fmt.Printf("%s", msg) //debug
			_, err := conn.Write(msg)
			if err != nil {
				panic(err)
			}
		}(out)
	}
}

//TODO use db query to get nick/user for prefix?
func (c *Client) sendReply(prefix, command string, params ...interface{}) error {
	var buf bytes.Buffer
	var ok bool = true
	var err error

	if prefix != "" {
		buf.WriteString(prefix)
		buf.WriteByte(' ')
	} else {
		ok = false
		buf.WriteString("<-")
	}

	if command != "" {
		buf.WriteString(command)
	} else {
		ok = false
		buf.WriteString("<-")
	}

	for _, p := range params {
		if s, ok := p.(string); ok {
			buf.WriteByte(' ')
			_, err = buf.WriteString(s)
			if err != nil {
				ok = false
				buf.WriteString("<-")
			}
		} else if a, ok := p.([]string); ok {
			for _, s := range a {
				buf.WriteByte(' ')
				_, err = buf.WriteString(s)
				if err != nil {
					ok = false
					buf.WriteString("<-")
				}
			}
		} else {
			ok = false
			buf.WriteString("<-")
		}
	}

	buf.WriteString(irc.CRLF)

	if !ok {
		return fmt.Errorf("Bad reply: %s", buf.String())
	}
	reply := buf.Bytes()
	outgoing <- &Reply{c, &reply}
	return nil
}

//TODO should this accept more than 0 or 2 args?
func getPrefix(args ...string) string {
	var prefix bytes.Buffer
	prefix.WriteByte(':')
	if len(args) == 2 {
		prefix.WriteString(args[0])
		prefix.WriteByte('!')
		prefix.WriteString(args[1])
		prefix.WriteByte('@')
	} else if len(args) != 0 {
		panic("Incorrect usage of getPrefix")
	}
	prefix.WriteString(irc.HOST_IP)
	return prefix.String()
}

// generic message handler
func (r *Received) handle() {
	switch strings.ToUpper(r.msg.Command) {
	case "QUIT":
		r.client.handleQuit()
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
	if r.msg.Params.Num < 1 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NEEDMOREPARAMS)
		if err != nil {
			irc.Debug(err)
		}
		return
	}

	// Query database for password
	password, ok := r.client.id.GetPassword()
	if !ok {
		// NO USER WITH THAT ID. DO SOMETHING
		return
	}

	if password != "" {
		err := r.client.sendReply(getPrefix(), irc.ERR_ALREADYREGISTRED)
		if err != nil {
			irc.Debug(err)
		}
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
		irc.Debug(nickname)
	}

	if r.msg.Params.Num < 1 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NONICKNAMEGIVEN)
		if err != nil {
			irc.Debug(err)
		}
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

	username, _, ok := r.client.id.GetUsernameRealname()
	if !ok {
		// NO USER WITH THAT ID
		return
	}

	if r.msg.Params.Num < 4 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NEEDMOREPARAMS)
		if err != nil {
			irc.Debug(err)
		}
		return
	}
	if username != "" {
		err := r.client.sendReply(getPrefix(), irc.ERR_ALREADYREGISTRED)
		if err != nil {
			irc.Debug(err)
		}
		return
	}

	//TODO probably check that each field is safe
	mode, err := strconv.Atoi(r.msg.Params.Others[1])
	if err != nil {
		log.Println(err)
		mode = 0
	}

	r.client.id.SetUsernameModeRealname(r.msg.Params.Others[0], r.msg.Params.Others[3], mode)
	err = r.client.sendReply(getPrefix(), irc.RPL_WELCOME, "Welcome to the Internet Relay Network!")
	if err != nil {
		irc.Debug(err)
	}
}

// Handles QUIT commands by removing session record from database.
func (client *Client) handleQuit() {
	delete(sessions, client.id)
	database.DeleteUser(client.id)
	//TODO probably close conn or signal channel exit
	//TODO also remove all other related database records & send appropriate messages
}

// Sends a message to target, which should be the first parameter. Target is
// either a nick, user, or channel.
func (r *Received) handlePrivMsg() {
	var recipient string

	nickname, username, ok := r.client.id.GetNicknameUsername()
	if !ok {
		return
	}

	// make sure there is a target and message to send
	if r.msg.Params.Num < 1 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NORECIPIENT, ":No recipient for", r.msg.Command)
		if err != nil {
			irc.Debug(err)
		}
		return
	}
	if r.msg.Params.Num < 2 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NOTEXTTOSEND, ":No text to send")
		if err != nil {
			irc.Debug(err)
		}
		return
	}

	//should this message be broadcast to a channel?
	recipient = r.msg.Params.Others[0]
	irc.Debug("Trying to send to", recipient)
	if message.IsChannel(recipient) {
		if !database.Check(recipient) {
			err := r.client.sendReply(getPrefix(), irc.ERR_NOSUCHCHANNEL, recipient, ":No such channel")
			if err != nil {
				irc.Debug(err)
			}
			return
		}
		if !r.client.id.IsMember(recipient) {
			err := r.client.sendReply(getPrefix(), irc.ERR_CANNOTSENDTOCHAN, recipient, ":Cannot send to channel")
			if err != nil {
				irc.Debug(err)
			}
			return
		}
		prefix := getPrefix(nickname, username)
		members := database.GetChannelUsers(r.msg.Params.Others[0])
		for _, userid := range members {
			target, ok := sessions[userid]
			if !ok {
				// probably worry about this, idk
				continue
			}
			err := target.sendReply(prefix, r.msg.Command, r.msg.Params.Others)
			if err != nil {
				irc.Debug(err)
			}
		}
	} else {
		// not a channel message, try to send PM to target user
		var targetid database.Id
		targetid, ok := database.GetIdByNickname(r.msg.Params.Others[0])
		if !ok {
			err := r.client.sendReply(getPrefix(), irc.ERR_NOSUCHNICK, r.msg.Params.Others[0], ":No such nick")
			if err != nil {
				irc.Debug(err)
			}
			return
		}

		targetSession, ok := sessions[targetid]
		if !ok {
			//target isn't in the system!! HALP!
			return
		}
		prefix := getPrefix(nickname, username)
		err := targetSession.sendReply(prefix, r.msg.Command, r.msg.Params.Others)
		if err != nil {
			irc.Debug(err)
		}
	}
}

func (r *Received) handleJoin() {
	var err error

	if r.msg.Params.Num < 1 {
		err = r.client.sendReply(getPrefix(), irc.ERR_NEEDMOREPARAMS)
		if err != nil {
			irc.Debug(err)
		}
		return
	}
	topic := database.JoinChannel(r.msg.Params.Others[0], r.client.id)
	nickname, username, _ := r.client.id.GetNicknameUsername()
	prefix := getPrefix(nickname, username)
	members := database.GetChannelUsers(r.msg.Params.Others[0])
	for _, m := range members {
		client := sessions[m]
		err = client.sendReply(prefix, r.msg.Command, r.msg.Params.Others[0])
		if err != nil {
			irc.Debug(err)
		}
	}

	if topic == "" {
		err = r.client.sendReply(getPrefix(), irc.RPL_NOTOPIC, r.msg.Params.Others[0], ":No topic set")
		if err != nil {
			irc.Debug(err)
		}
	} else {
		err = r.client.sendReply(getPrefix(), irc.RPL_TOPIC, r.msg.Params.Others[0], ':', topic)
		if err != nil {
			irc.Debug(err)
		}
	}
	//TODO implement RPL_NAMREPLY
}

func (r *Received) handlePart() {
	if r.msg.Params.Num < 1 {
		err := r.client.sendReply(getPrefix(), irc.ERR_NEEDMOREPARAMS)
		if err != nil {
			irc.Debug(err)
		}
		return
	}
	if !database.Check(r.msg.Params.Others[0]) {
		err := r.client.sendReply(getPrefix(), irc.ERR_NOSUCHCHANNEL, r.msg.Params.Others[0], ":No such channel")
		if err != nil {
			irc.Debug(err)
		}
		return
	}
	if !r.client.id.IsMember(r.msg.Params.Others[0]) {
		err := r.client.sendReply(getPrefix(), irc.ERR_NOTONCHANNEL, r.msg.Params.Others[0], ":Not on channel")
		if err != nil {
			irc.Debug(err)
		}
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
