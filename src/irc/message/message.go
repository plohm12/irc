package message

import (
	"bytes"
	"strconv"
	"strings"
)

type Prefix string
type Param string
type Command string
type Message struct {
	Prefix
	Command
	Params []Param
}

func NewMessage() Message {
	return Message{}
}

func (c Command) ToUpper() string {
	var s string = string(c)
	return strings.ToUpper(s)
}

func (p Param) IsChannel() bool {
	switch p[0] {
	case '#':
		return true
	case '&':
		return true
	case '+':
		return true
	case '!':
		return true
	}
	return false
}

func (p Param) ToInt() (int, error) {
	var s string = string(p)
	return strconv.Atoi(s)
}

func (p Prefix) ToString() string {
	return string(p)
}

func (c Command) ToString() string {
	return string(c)
}

func (p Param) ToString() string {
	return string(p)
}

func (m *Message) ToString() string {
	var buf bytes.Buffer

	if m.Prefix != "" {
		buf.WriteString("(Prefix:")
		buf.WriteString(string(m.Prefix))
		buf.WriteByte(')')
	}
	if m.Command != "" {
		buf.WriteString("(Command:")
		buf.WriteString(string(m.Command))
		buf.WriteByte(')')
	}
	for i, p := range m.Params {
		buf.WriteString("(Param[")
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString("]:")
		buf.WriteString(string(p))
		buf.WriteByte(')')
	}

	return buf.String()
}
