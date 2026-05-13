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

func NewParser(s string) Parser {
	return Parser{buf: s, pos: 0}
}

type NamedCell struct {
	column string
	value  Cell
}
type StmtSelect struct {
	table string
	cols  []string
	keys  []NamedCell
}
type StmtCreateTable struct {
	table string
	cols  []Column
	pkey  []string
}
type StmtInsert struct {
	table string
	value []Cell
}
type StmtUpdate struct {
	table string
	keys  []NamedCell
	value []NamedCell
}
type StmtDelete struct {
	table string
	keys  []NamedCell
}

func isSpace(ch byte) bool {
	switch ch {
	case '\t', '\n', '\v', '\f', '\r', ' ':
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
	return isAlpha(ch) || isDigit(ch) || ch == '_'
}
func isSeparator(ch byte) bool {
	return ch < 128 && !isNameContinue(ch)
}

func (p *Parser) skipSpaces() {
	for p.pos < len(p.buf) && isSpace(p.buf[p.pos]) {
		p.pos++
	}
}
func (p *Parser) tryKeyword(kw ...string) bool {
	save := p.pos
	for _, kw := range kw {
		p.skipSpaces()
		if !(p.pos+len(kw) <= len(p.buf) && strings.EqualFold(p.buf[p.pos:p.pos+len(kw)], kw)) {
			p.pos = save
			return false
		}
		if p.pos+len(kw) < len(p.buf) && !isSeparator(p.buf[p.pos+len(kw)]) {
			p.pos = save
			return false
		}
		p.pos += len(kw)
	}
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
func (p *Parser) parseString(out *Cell) error {
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
				return errors.New("invalid escape sequence")
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
	return errors.New("string is not terminated")
}
func (p *Parser) parseInt(out *Cell) (err error) {
	start, curr := p.pos, p.pos
	if p.buf[curr] == '-' || p.buf[curr] == '+' {
		curr++
	}
	for curr < len(p.buf) && isDigit(p.buf[curr]) {
		curr++
	}
	if out.I64, err = strconv.ParseInt(string(p.buf[start:curr]), 10, 64); err != nil {
		return err
	}
	out.Type = TypeI64
	p.pos = curr
	return nil
}
func (p *Parser) parseValue(out *Cell) error {
	p.skipSpaces()
	if p.pos >= len(p.buf) {
		return errors.New("expecting value")
	}
	ch := p.buf[p.pos]
	if ch == '"' || ch == '\'' {
		return p.parseString(out)
	} else if isDigit(ch) || ch == '-' || ch == '+' {
		return p.parseInt(out)
	} else {
		return errors.New("expecting value")
	}
}
func (p *Parser) parseEqual(out *NamedCell) error {
	var ok bool
	out.column, ok = p.tryName()
	if !ok {
		return errors.New("expecting column")
	}
	if !p.tryPunctuation("=") {
		return errors.New("expecting equal")
	}
	return p.parseValue(&out.value)
}
func (p *Parser) parseSelect(out *StmtSelect) error {
	for !p.tryKeyword("FROM") {
		if len(out.cols) > 0 && !p.tryPunctuation(",") {
			return errors.New("expecting comma")
		}
		if name, ok := p.tryName(); ok {
			out.cols = append(out.cols, name)
		} else {
			return errors.New("expecting column")
		}
	}
	if len(out.cols) == 0 {
		return errors.New("expecting column")
	}
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expecting table")
	}
	return p.parseWhere(&out.keys)
}
func (p *Parser) parseWhere(out *[]NamedCell) error {
	if !p.tryKeyword("WHERE") {
		return errors.New("expecting WHERE")
	}
	for !p.tryPunctuation(";") {
		expr := NamedCell{}
		if len(*out) > 0 && !p.tryKeyword("AND") {
			return errors.New("expecting AND")
		}
		if err := p.parseEqual(&expr); err != nil {
			return err
		}
		*out = append(*out, expr)
	}
	if len(*out) == 0 {
		return errors.New("expecting where clause")
	}
	return nil
}
func (p *Parser) parseCommaList(item func() error) error {
	if !p.tryPunctuation("(") {
		return errors.New("expect (")
	}
	comma := false
	for !p.tryPunctuation(")") {
		if comma && !p.tryPunctuation(",") {
			return errors.New("expect , comma")
		}
		comma = true
		if err := item(); err != nil {
			return err
		}
	}
	return nil
}
func (p *Parser) parseValueItem(out *[]Cell) error {
	cell := Cell{}
	if err := p.parseValue(&cell); err != nil {
		return err
	}
	*out = append(*out, cell)
	return nil
}
func (p *Parser) parseNameItem(out *[]string) error {
	name, ok := p.tryName()
	if !ok {
		return errors.New("expecting name")
	}
	*out = append(*out, name)
	return nil
}
func (p *Parser) parseInsert(out *StmtInsert) error {
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expecting table")
	}
	if !p.tryKeyword("VALUES") {
		return errors.New("expecting VALUES")
	}
	err := p.parseCommaList(func() error { return p.parseValueItem(&out.value) })
	if err != nil {
		return err
	}
	if !p.tryPunctuation(";") {
		return errors.New("expect ; semicolon")
	}
	return nil
}
func (p *Parser) parseUpdate(out *StmtUpdate) error {
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expect table name")
	}
	if !p.tryKeyword("SET") {
		return errors.New("expect SET")
	}
	for !p.tryKeyword("WHERE") {
		expr := NamedCell{}
		if len(out.value) > 0 && !p.tryKeyword(",") {
			return errors.New("expect , comma")
		}
		if err := p.parseEqual(&expr); err != nil {
			return err
		}
		out.value = append(out.value, expr)
	}
	if len(out.value) == 0 {
		return errors.New("expect assignment list")
	}
	p.pos -= len("WHERE")
	return p.parseWhere(&out.keys)
}
func (p *Parser) parseDelete(out *StmtDelete) error {
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expecting table name")
	}
	return p.parseWhere(&out.keys)
}
func (p *Parser) parseCreateTableItem(out *StmtCreateTable) error {
	if p.tryKeyword("PRIMARY", "KEY") {
		return p.parseCommaList(func() error { return p.parseNameItem(&out.pkey) })
	}
	var ok bool
	col := Column{}
	if col.Name, ok = p.tryName(); !ok {
		return errors.New("expecting name")
	}
	kind, ok := p.tryName()
	if !ok {
		return errors.New("expecting name")
	}
	switch kind {
	case "int64":
		col.Type = TypeI64
	case "string":
		col.Type = TypeStr
	default:
		return errors.New("unknown column type")
	}
	out.cols = append(out.cols, col)
	return nil
}
func (p *Parser) parseCreateTable(out *StmtCreateTable) error {
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expecting table name")
	}
	err := p.parseCommaList(func() error { return p.parseCreateTableItem(out) })
	if err != nil {
		return err
	}
	if !p.tryPunctuation(";") {
		return errors.New("expect ; semicolon")
	}
	return nil
}
func (p *Parser) parseStmt() (out interface{}, err error) {
	if p.tryKeyword("SELECT") {
		stmt := &StmtSelect{}
		err = p.parseSelect(stmt)
		out = stmt
	} else if p.tryKeyword("CREATE", "TABLE") {
		stmt := &StmtCreateTable{}
		err = p.parseCreateTable(stmt)
		out = stmt
	} else if p.tryKeyword("INSERT", "INTO") {
		stmt := &StmtInsert{}
		err = p.parseInsert(stmt)
		out = stmt
	} else if p.tryKeyword("UPDATE") {
		stmt := &StmtUpdate{}
		err = p.parseUpdate(stmt)
		out = stmt
	} else if p.tryKeyword("DELETE", "FROM") {
		stmt := &StmtDelete{}
		err = p.parseDelete(stmt)
		out = stmt
	} else {
		err = errors.New("unknown statement")
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}
func (p *Parser) isEnd() bool {
	p.skipSpaces()
	return p.pos >= len(p.buf)
}
