package Database

import (
	"errors"
	"strconv"
	"strings"
)

type Parser struct {
	buf string
	pos int
}
type StmtSelect struct {
	table string
	cols  []string
	keys  []NamedCell
}

type NamedCell struct {
	column string
	value  Cell
}

func NewParser(s string) Parser {
	return Parser{buf: s, pos: 0}
}

func (p *Parser) isEnd() bool {
	p.skipSpaces()
	return p.pos >= len(p.buf)
}
func (p *Parser) skipSpaces() {
	for p.pos < len(p.buf) && isSpace(p.buf[p.pos]) {
		p.pos++
	}
}
func isSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}
func isAlpha(ch byte) bool {
	return 'a' <= (ch|32) && (ch|32) <= 'z'
}
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
func isNameStart(ch byte) bool {
	return isAlpha(ch) || ch == '_'
}
func isNameContinue(ch byte) bool {
	return isNameStart(ch) || isDigit(ch)
}
func isSeparator(ch byte) bool {
	return ch < 128 && !isNameContinue(ch)
}
func (p *Parser) tryName() (string, bool) {
	p.skipSpaces()
	start, curr := p.pos, p.pos
	if !(curr < len(p.buf) && isNameStart(p.buf[curr])) {
		return "", false
	}
	curr++
	for curr < len(p.buf) && isNameContinue(p.buf[curr]) {
		curr++
	}
	p.pos = curr
	return p.buf[start:curr], true
}
func (p *Parser) tryKeyword(kw string) bool {
	p.skipSpaces()
	if !(p.pos+len(kw) <= len(p.buf) && strings.EqualFold(p.buf[p.pos:p.pos+len(kw)], kw)) {
		return false
	}
	if p.pos+len(kw) < len(p.buf) && !isSeparator(p.buf[p.pos+len(kw)]) {
		return false
	}
	p.pos += len(kw)
	return true
}
func (p *Parser) tryPunctuation(tok string) bool {
	p.skipSpaces()
	if !(p.pos+len(tok) <= len(p.buf) && p.buf[p.pos:p.pos+len(tok)] == tok) {
		return false
	}
	p.pos += len(tok)
	return true
}
func (p *Parser) parseValue(out *Cell) error {
	p.skipSpaces()
	if p.pos >= len(p.buf) {
		return errors.New("expected value")
	}
	ch := p.buf[p.pos]
	if ch == '"' || ch == '\'' {
		return p.parseString(out)
	} else if isDigit(ch) || ch == '-' || ch == '+' {
		return p.parseInt(out)
	}
	return errors.New("expected value")
}
func (p *Parser) parseInt(out *Cell) (err error) {
	start, curr := p.pos, p.pos
	if p.buf[curr] == '-' || p.buf[curr] == '+' {
		curr++
	}
	for curr < len(p.buf) && isDigit(p.buf[curr]) {
		curr++
	}
	if out.I64, err = strconv.ParseInt(p.buf[start:curr], 10, 64); err != nil {
		return err
	}
	out.Type = TypeI64
	p.pos = curr
	return nil
}
func (p *Parser) parseString(out *Cell) (err error) {
	quote := p.buf[p.pos]
	curr := p.pos + 1
	for curr < len(p.buf) {
		ch := p.buf[curr]
		if ch == '\\' {
			curr++
			if curr < len(p.buf) && (p.buf[curr] == '"' || p.buf[curr] == '\'') {
				out.Str = append(out.Str, p.buf[curr])
				curr++
			} else {
				return errors.New("bad escape")
			}
		} else if ch == quote {
			out.Type = TypeStr
			p.pos = curr + 1
			return nil
		} else {
			out.Str = append(out.Str, p.buf[curr])
			curr++
		}
	}
	return errors.New("string not terminated")
}
func (p *Parser) parseEqual(out *NamedCell) error {
	var ok bool
	out.column, ok = p.tryName()
	if !ok {
		return errors.New("expected column")
	}
	if !p.tryPunctuation("=") {
		return errors.New("expected =")
	}
	return p.parseValue(&out.value)
}
func (p *Parser) parseSelect(out *StmtSelect) error {
	if !p.tryKeyword("SELECT") {
		return errors.New("expected keyword")
	}
	for !p.tryKeyword("FROM") {
		if len(out.cols) > 0 && !p.tryPunctuation(",") {
			return errors.New("expected comma")
		}
		if name, ok := p.tryName(); ok {
			out.cols = append(out.cols, name)
		} else {
			return errors.New("expected column")
		}
	}
	if len(out.cols) == 0 {
		return errors.New("expected columns list")
	}
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expected table name")
	}
	return p.parseWhere(&out.keys)
}
func (p *Parser) parseWhere(out *[]NamedCell) error {
	if !p.tryKeyword("WHERE") {
		return errors.New("expected keyword")
	}
	for !p.tryPunctuation(";") {
		expr := NamedCell{}
		if len(*out) > 0 && !p.tryKeyword("AND") {
			return errors.New("expected AND")
		}
		if err := p.parseEqual(&expr); err != nil {
			return err
		}
		*out = append(*out, expr)
	}
	if len(*out) == 0 {
		return errors.New("expect where clause")
	}
	return nil
}
