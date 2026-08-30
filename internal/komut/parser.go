package komut

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Invocation struct {
	Commands []Command
	Lead     string
	HasLead  bool
}

type Command struct {
	Name string
	Args []string
}

func Parse(input string) (Invocation, error) {
	if !utf8.ValidString(input) {
		return Invocation{}, fail(ErrInvalidInvocation, "", "invocation is not valid UTF-8")
	}

	p := parser{input: input}
	p.skipSpace()

	if !p.consumeLiteral("$x") || p.eof() || !p.spaceAt(p.pos) {
		return Invocation{}, fail(ErrInvalidInvocation, "", "expected $x followed by whitespace")
	}
	p.skipSpace()
	if p.eof() {
		return Invocation{}, fail(ErrInvalidInvocation, "", "missing command")
	}

	var invocation Invocation
	for {
		if p.specialToken("--") {
			return Invocation{}, fail(ErrInvalidInvocation, "", "lead marker requires a command before it")
		}
		if p.specialToken("+") {
			return Invocation{}, fail(ErrInvalidInvocation, "", "missing command before +")
		}

		command, err := p.parseCommand()
		if err != nil {
			return Invocation{}, err
		}
		invocation.Commands = append(invocation.Commands, command)

		p.skipSpace()
		if p.eof() {
			return invocation, nil
		}
		if p.specialToken("--") {
			p.pos += 2
			p.skipSpace()
			invocation.HasLead = true
			invocation.Lead = p.input[p.pos:]
			return invocation, nil
		}
		if p.specialToken("+") {
			p.pos++
			p.skipSpace()
			if p.eof() {
				return Invocation{}, fail(ErrInvalidInvocation, "", "missing command after +")
			}
			continue
		}

		return Invocation{}, fail(ErrInvalidInvocation, "", "unexpected parser state")
	}
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseCommand() (Command, error) {
	name, err := p.parseBare()
	if err != nil {
		return Command{}, err
	}
	if !ValidCommandName(name) {
		return Command{}, fail(ErrInvalidCommand, name, "invalid command name")
	}

	command := Command{Name: name}
	for {
		p.skipSpace()
		if p.eof() || p.specialToken("+") || p.specialToken("--") {
			return command, nil
		}

		arg, err := p.parseArgument()
		if err != nil {
			return Command{}, err
		}
		command.Args = append(command.Args, arg)
	}
}

func (p *parser) parseArgument() (string, error) {
	if p.eof() {
		return "", fail(ErrInvalidInvocation, "", "missing argument")
	}

	switch p.input[p.pos] {
	case '\'':
		return p.parseSingleQuoted()
	case '"':
		return p.parseDoubleQuoted()
	default:
		return p.parseBare()
	}
}

func (p *parser) parseBare() (string, error) {
	start := p.pos
	for !p.eof() && !p.spaceAt(p.pos) {
		if p.input[p.pos] == '\'' || p.input[p.pos] == '"' {
			return "", fail(ErrInvalidInvocation, "", "quotes must begin an argument")
		}
		_, size := utf8.DecodeRuneInString(p.input[p.pos:])
		if size == 0 {
			break
		}
		p.pos += size
	}
	if start == p.pos {
		return "", fail(ErrInvalidInvocation, "", "expected token")
	}
	return p.input[start:p.pos], nil
}

func (p *parser) parseSingleQuoted() (string, error) {
	p.pos++
	start := p.pos
	for !p.eof() {
		if p.input[p.pos] == '\'' {
			value := p.input[start:p.pos]
			p.pos++
			if !p.eof() && !p.spaceAt(p.pos) {
				return "", fail(ErrInvalidInvocation, "", "quoted argument must end at a token boundary")
			}
			return value, nil
		}
		_, size := utf8.DecodeRuneInString(p.input[p.pos:])
		p.pos += size
	}
	return "", fail(ErrUnterminatedQuote, "", "unterminated single quote")
}

func (p *parser) parseDoubleQuoted() (string, error) {
	p.pos++
	var b strings.Builder
	for !p.eof() {
		switch p.input[p.pos] {
		case '"':
			p.pos++
			if !p.eof() && !p.spaceAt(p.pos) {
				return "", fail(ErrInvalidInvocation, "", "quoted argument must end at a token boundary")
			}
			return b.String(), nil
		case '\\':
			if p.pos+1 < len(p.input) {
				next := p.input[p.pos+1]
				if next == '\\' || next == '"' {
					b.WriteByte(next)
					p.pos += 2
					continue
				}
			}
			b.WriteByte('\\')
			p.pos++
		default:
			r, size := utf8.DecodeRuneInString(p.input[p.pos:])
			b.WriteRune(r)
			p.pos += size
		}
	}
	return "", fail(ErrUnterminatedQuote, "", "unterminated double quote")
}

func (p *parser) specialToken(token string) bool {
	if !strings.HasPrefix(p.input[p.pos:], token) {
		return false
	}
	end := p.pos + len(token)
	return end == len(p.input) || p.spaceAt(end)
}

func (p *parser) consumeLiteral(value string) bool {
	if !strings.HasPrefix(p.input[p.pos:], value) {
		return false
	}
	p.pos += len(value)
	return true
}

func (p *parser) skipSpace() {
	for !p.eof() && p.spaceAt(p.pos) {
		_, size := utf8.DecodeRuneInString(p.input[p.pos:])
		p.pos += size
	}
}

func (p *parser) spaceAt(pos int) bool {
	if pos >= len(p.input) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(p.input[pos:])
	return unicode.IsSpace(r)
}

func (p *parser) eof() bool {
	return p.pos >= len(p.input)
}

func ValidCommandName(name string) bool {
	if name == "" || len(name) > 64 || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || strings.Contains(name, "//") {
		return false
	}

	for segment := range strings.SplitSeq(name, "/") {
		if !validSegment(segment) {
			return false
		}
	}
	return true
}

func validSegment(segment string) bool {
	if segment == "" || strings.Contains(segment, "--") || !isAlphaNum(segment[0]) || !isAlphaNum(segment[len(segment)-1]) {
		return false
	}
	for i := range len(segment) {
		c := segment[i]
		if !(isAlphaNum(c) || c == '-') {
			return false
		}
	}
	return true
}

func isAlphaNum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func HasInvocationPrefix(input string) bool {
	if !utf8.ValidString(input) {
		return false
	}
	p := parser{input: input}
	p.skipSpace()
	if !p.consumeLiteral("$x") || p.eof() {
		return false
	}
	return p.spaceAt(p.pos)
}
