package parser

import(
	"bufio"
	"bytes"
	"io"
)

type Token byte

const(
	EOF = 0
	SPACE = 1
	COLON = 2
	ASTERISK = 3
	CRLF = 4
	BANG = 6
	AT = 7
	COMMA = 8
	PERCENT = 9
	HASH = 10
	PLUS = 11
	AND = 12
	PERIOD = 13
	DASH = 14
	DOLLAR = 15
	BELL = 16
	FF = 17
	TAB = 18
	VT = 19
	LETTER = 20
	DIGIT = 21
	SPECIAL = 22
	OTHER = 23
	ILLEGAL = 24
)

var eof = rune(0)

func (s *Scanner) Scan() (tok Token, lit string) {
	ch := s.read()

	switch ch {
	case eof:
		return EOF, ""
	case ' ':
		return SPACE, string(ch)
	case ':':
		return COLON, string(ch)
	case '*':
		return ASTERISK, string(ch)
	case '\r':
		s.unread()
		return s.scanCrlf()
	case '!':
		return BANG, string(ch)
	case '@':
		return AT, string(ch)
	case ',':
		return COMMA, string(ch)
	case '%':
		return PERCENT, string(ch)
	case '#':
		return HASH, string(ch)
	case '+':
		return PLUS, string(ch)
	case '&':
		return AND, string(ch)
	case '.':
		return PERIOD, string(ch)
	case '-':
		return DASH, string(ch)
	case '$':
		return DOLLAR, string(ch)
	case '\x07':
		return BELL, string(ch)
	case '\x0c':
		return FF, string(ch)
	case '\t':
		return TAB, string(ch)
	case '\x0b':
		return VT, string(ch)
	}

	if isLetter(ch) {
		return LETTER, string(ch)
	} else if isDigit(ch) {
		return DIGIT, string(ch)
	} else if isSpecial(ch) {
		return SPECIAL, string(ch)
	} else {
		return OTHER, string(ch)
	}
}

func (s *Scanner) scanCrlf() (Token, string) {
	var crlf bytes.Buffer

	ch := s.read()
	crlf.WriteRune(ch)
	ch = s.read()
	crlf.WriteRune(ch)
	if ch != '\n' {
		return ILLEGAL, crlf.String()
	} else {
		return CRLF, crlf.String()
	}
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isSpecial(ch rune) bool {
	switch ch {
	case '[':
	case ']':
	case '\\':
	case '`':
	case '_':
	case '^':
	case '{':
	case '}':
	case '|':
		return true
	}
	return false
}

type Scanner struct {
	r *bufio.Reader
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
}
