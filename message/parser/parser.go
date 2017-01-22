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

func (p *Parser) scanPrefix() (*message.Prefix, error) {
	prefix := message.NewPrefix()

	serverName, err := p.scanServerName()
	if err == nil {
		prefix.ServerName = serverName
		return &prefix, nil
	} else if prefix.Nickname, err = p.scanNickname(); err == nil {
		if prefix.User, prefix.Host, err = p.scanPrefixPrime(); err == nil {
			return &prefix, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (p *Parser) scanPrefixPrime() (user string, host *message.Host, err error) {
	if tok, _ := p.scan(); tok == BANG {
		if user, err = p.scanUser(); err != nil {
			return "", nil, err
		}
	}

	if host, err := p.scanPrefixPrimePrime(); err != nil {
		return "", nil, err
	} else {
		return user, host, nil
	}
}

func (p *Parser) scanPrefixPrimePrime() (*message.Host, error) {
	if tok, _ := p.scan(); tok != AT {
		p.unscan()
		return nil, nil
	}
	return p.scanHost()
}

func (p *Parser) scanCommand() (string, error) {
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
				return command.String(), nil
			}
		}
	} else if tok == LETTER {
		command.WriteString(lit)
		for {
			if tok, lit := p.scan(); tok != LETTER {
				p.unscan()
				return command.String(), nil
			} else {
				command.WriteString(lit)
			}
		}
	} else {
		return "", fmt.Errorf("Scanning command found %q, expected DIGIT or LETTER", tok)
	}
}

func (p *Parser) scanParams() (*message.Params, error) {
	params := message.NewParams()
	params.Num = 0

	for i := 0; i < 15; i++ {
		var param bytes.Buffer
		if tok, _ := p.scan(); tok != SPACE {
			p.unscan()
			return &params, nil
		}

		if tok, lit := p.scan(); tok == COLON {
			for {
				if tok, lit := p.scan(); tok == CRLF {
					p.unscan()
					params.Others = append(params.Others, param.String())
					params.Num++
					return &params, nil
				} else {
					param.WriteString(lit)
				}
			}
		} else if tok == CRLF {
			p.unscan()
			return &params, nil
		} else {
			param.WriteString(lit)
		}

		for {
			if tok, lit := p.scan(); tok == CRLF || tok == SPACE {
				p.unscan()
				params.Others = append(params.Others, param.String())
				params.Num++
				break
			} else {
				param.WriteString(lit)
			}
		}
	}

	return &params, nil
}

func (p *Parser) scanServerName() (string, error) {
	return p.scanHostName()
}

func (p *Parser) scanHost() (*message.Host, error) {
	host := message.NewHost()

	hostName, err := p.scanHostName()
	if err == nil {
		host.HostName = hostName
		return &host, nil
	} else {
		hostAddr, err := p.scanHostAddr()
		if err == nil {
			host.HostAddr = hostAddr
			return &host, nil
		} else {
			return nil, err
		}
	}
}

func (p *Parser) scanNickname() (string, error) {
	var nick bytes.Buffer

	if tok, lit := p.scan(); tok != LETTER && tok != SPECIAL {
		return "", fmt.Errorf("Scanning nickname found %q, expected LETTER or SPECIAL", tok)
	} else {
		nick.WriteString(lit)
	}

	for i := 1; i < 9; i++ {
		if tok, lit := p.scan(); tok != LETTER && tok != DIGIT && tok != DASH && tok != SPECIAL {
			return "", fmt.Errorf("Scanning nickname found %q, expected LETTER, DIGIT, DASH, or SPECIAL", tok)
		} else {
			nick.WriteString(lit)
		}
	}
	return nick.String(), nil
}

func (p *Parser) scanHostName() (string, error) {
	var host bytes.Buffer

	for {
		if tok, lit := p.scan(); tok != LETTER && tok != DIGIT {
			return "", fmt.Errorf("Scanning Hostname found %q, expected LETTER or DIGIT", tok)
		} else {
			host.WriteString(lit)
		}

		for {
			tok, lit := p.scan()
			if tok == PERIOD {
				host.WriteString(lit)
				break
			} else if tok == LETTER || tok == DIGIT {
				host.WriteString(lit)
			} else if tok == DASH {
				host.WriteString(lit)
				if tok, lit = p.scan(); tok != LETTER && tok != DIGIT {
					return "", fmt.Errorf("Scanning Hostname found %q, expected LETTER or DIGIT following DASH", tok)
				} else {
					host.WriteString(lit)
				}
			} else {
				return host.String(), nil
			}
		}
	}
}

// Only IPv4 implemented for now
func (p *Parser) scanHostAddr() (string, error) {
	var addr bytes.Buffer

	for i := 0; i < 4; i++ {
		if tok, lit := p.scan(); tok != DIGIT {
			return "", fmt.Errorf("Scanning host address found %q, expected DIGIT", tok)
		} else {
			addr.WriteString(lit)
		}

		if tok, lit := p.scan(); tok == PERIOD {
			addr.WriteString(lit)
			continue
		} else if tok == DIGIT {
			addr.WriteString(lit)
		} else {
			return "", fmt.Errorf("Scanning host address found %q, expected DIGIT or PERIOD", tok)
		}

		if tok, lit := p.scan(); tok == PERIOD {
			addr.WriteString(lit)
			continue
		} else if tok == DIGIT {
			addr.WriteString(lit)
		} else {
			return "", fmt.Errorf("Scanning host address found %q, expected DIGIT or PERIOD", tok)
		}
	}

	return addr.String(), nil
}

func (p *Parser) scanUser() (string, error) {
	var user bytes.Buffer

	if tok, lit := p.scan(); tok == SPACE || tok == AT || tok == CRLF {
		return "", fmt.Errorf("Scanning User found %q, expected anything else", tok)
	} else {
		user.WriteString(lit)
	}

	for {
		if tok, lit := p.scan(); tok == SPACE || tok == AT || tok == CRLF {
			p.unscan()
			return user.String(), nil
		} else {
			user.WriteString(lit)
		}
	}
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
