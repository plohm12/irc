//TODO get rid of the Params.Others bullshit
//TODO (p Param) IsChannel() instead of current implementation

package message

import (
	"bytes"
	"fmt"
)

type Symbol interface {
	String() string
}

func Print(s Symbol) {
	fmt.Printf("(%T:%v)\n", s, s.String())
}

type Message struct {
	*Prefix
	Command string
	*Params
}

func NewMessage() Message {
	return Message{}
}

func NewPrefix() Prefix {
	return Prefix{}
}

func NewParams() Params {
	return Params{}
}

func NewHost() Host {
	return Host{}
}

func IsChannel(param string) bool {
	switch param[0] {
	case '#':
	case '&':
	case '+':
	case '!':
		return true
	}
	return false
}

func (m *Message) String() string {
	var buf bytes.Buffer

	if m.Prefix != nil {
		buf.WriteString(fmt.Sprintf("(Prefix:%v)", m.Prefix.String()))
	}
	if m.Command != "" {
		buf.WriteString(fmt.Sprintf("(Command:%v)", m.Command))
	}
	if m.Params != nil {
		buf.WriteString(fmt.Sprintf("(Params:%v)", m.Params.String()))
	}
	if buf.String() == "" {
		buf.WriteString("(No Prefix, Command, or Params)")
	}

	return buf.String()
}

type Prefix struct {
	ServerName string
	Nickname   string
	User       string
	*Host
}

func (p *Prefix) String() string {
	var buf bytes.Buffer

	if p.ServerName != "" {
		buf.WriteString(fmt.Sprintf("(ServerName:%v)", p.ServerName))
	}
	if p.Nickname != "" {
		buf.WriteString(fmt.Sprintf("(Nick:%v)", p.Nickname))
	}
	if p.User != "" {
		buf.WriteString(fmt.Sprintf("(User:%v)", p.User))
	}
	if p.Host != nil {
		buf.WriteString(fmt.Sprintf("(Host:%v)", p.Host.String()))
	}
	if buf.String() == "" {
		buf.WriteString("(No ServerName, Nick, User, or Host)")
	}

	return buf.String()
}

type Params struct {
	//Target string
	//*MsgTo
	Others []string
	Num    int
}

func (p *Params) String() string {
	var buf bytes.Buffer

	//if p.Target != "" {
	//	buf.WriteString(fmt.Sprintf("(Target:%v)", p.Target))
	//}
	for i, v := range p.Others {
		buf.WriteString(fmt.Sprintf("(Param[%d]:%v)", i, v))
	}
	if buf.String() == "" {
		buf.WriteString("(No Params)")
	}

	return buf.String()
}

type Host struct {
	HostName string
	HostAddr string
}

func (h *Host) String() string {
	var buf bytes.Buffer

	if h.HostName != "" {
		buf.WriteString(fmt.Sprintf("(HostName:%v)", h.HostName))
	}
	if h.HostAddr != "" {
		buf.WriteString(fmt.Sprintf("(HostAddr:%v)", h.HostAddr))
	}
	if buf.String() == "" {
		buf.WriteString("(No HostName or HostAddr)")
	}

	return buf.String()
}
