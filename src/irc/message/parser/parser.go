//TODO: Rename nested tok/lit's so its less confusing
//TODO: Add panic()'s for better error handling

package parser

import (
	"bytes"
	"fmt"
	"io"
	"irc/message"
)

func (p *Parser) Parse() (*message.Message, error) {
	msg := message.NewMessage()

	fmt.Println("Parsing...")

	// Check for message prefix
	if tok, _ := p.scan(); tok == COLON {
		p.unscan()
		prefix, err := p.scanPrefix()
		if err != nil {
			return nil, err
		} else {
			msg.Prefix = prefix
		}
	} else {
		p.unscan()
	}

	if tok, _ := p.scan(); tok != SPACE {
		p.unscan()
	}

	// Parse command
	command, err := p.scanCommand()
	if err != nil {
		return nil, err
	} else {
		msg.Command = command
	}

	// Check for message parameters
	msg.Params, _ = p.scanParams()

	if tok, lit := p.scan(); tok != CRLF {
		return nil, fmt.Errorf("Scanning message found %q, expected CRLF", lit)
	}

	return &msg, nil
}

func (p *Parser) scanPrefix() (message.Prefix, error) {
	var prefix bytes.Buffer

	for tok, lit := p.scan(); tok != SPACE; {
		prefix.WriteString(lit)
	}

	return message.Prefix(prefix.String()), nil
}

func (p *Parser) scanCommand() (message.Command, error) {
	var command bytes.Buffer

	if tok, lit := p.scan(); tok == DIGIT {
		command.WriteString(lit)
		if tok, lit = p.scan(); tok != DIGIT {
			return "", fmt.Errorf("Scanning command found %q, expected DIGIT", tok)
		} else {
			command.WriteString(lit)
			if tok, lit = p.scan(); tok != DIGIT {
				return "", fmt.Errorf("Scanning command found %q, expected DIGIT", tok)
			} else {
				command.WriteString(lit)
				return message.Command(command.String()), nil
			}
		}
	} else if tok == LETTER {
		command.WriteString(lit)
		for {
			if tok, lit := p.scan(); tok != LETTER {
				p.unscan()
				return message.Command(command.String()), nil
			} else {
				command.WriteString(lit)
			}
		}
	} else {
		return "", fmt.Errorf("Scanning command found %q, expected DIGIT or LETTER", tok)
	}
}

func (p *Parser) scanParams() ([]message.Param, error) {
	var params []message.Param

	for i := 0; i < 15; i++ {
		var param bytes.Buffer
		if tok, _ := p.scan(); tok != SPACE {
			p.unscan()
			return params, nil
		}

		if tok, lit := p.scan(); tok == COLON {
			for {
				if tok, lit := p.scan(); tok == CRLF {
					p.unscan()
					params = append(params, message.Param(param.String()))
					return params, nil
				} else {
					param.WriteString(lit)
				}
			}
		} else if tok == CRLF {
			p.unscan()
			return params, nil
		} else {
			param.WriteString(lit)
		}

		for {
			if tok, lit := p.scan(); tok == CRLF || tok == SPACE {
				p.unscan()
				params = append(params, message.Param(param.String()))
				break
			} else {
				param.WriteString(lit)
			}
		}
	}

	return params, nil
}

type Parser struct {
	s   *Scanner
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max = 1)
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

func (p *Parser) scan() (tok Token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.Scan()
	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *Parser) unscan() {
	p.buf.n = 1
}
