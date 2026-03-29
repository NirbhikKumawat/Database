package Database

import (
	"errors"
	"fmt"
)

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
type NamedCell struct {
	column string
	value  Cell
}

func (p *Parser) parseSelect(out *StmtSelect) error {
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
		return errors.New("expected column list")
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
		return errors.New("expected where clause")
	}
	return nil
}
func (p *Parser) parseCommaList(item func() error) error {
	if !p.tryPunctuation("(") {
		return errors.New("expected (")
	}
	comma := false
	for !p.tryPunctuation(")") {
		if comma && !p.tryPunctuation(",") {
			return errors.New("expected , ")
		}
		comma = true
		if err := item(); err != nil {
			return err
		}
	}
	return nil
}
func (p *Parser) parseNameItem(out *[]string) error {
	name, ok := p.tryName()
	if !ok {
		return errors.New("expected name")
	}
	*out = append(*out, name)
	return nil
}
func (p *Parser) parseCreateTableItem(out *StmtCreateTable) error {
	if p.tryKeyword("PRIMARY", "KEY") {
		return p.parseCommaList(func() error {
			return p.parseNameItem(&out.pkey)
		})
	}
	var ok bool
	col := Column{}
	if col.Name, ok = p.tryName(); !ok {
		return errors.New("expected name")
	}
	kind, ok := p.tryName()
	if !ok {
		return errors.New("expected name")
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
		return errors.New("expected table name")
	}
	err := p.parseCommaList(func() error {
		return p.parseCreateTableItem(out)
	})
	if err != nil {
		return err
	}
	if !p.tryKeyword(";") {
		return errors.New("expected ; ")
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
func (p *Parser) parseInsert(out *StmtInsert) error {
	var ok bool
	if out.table, ok = p.tryName(); !ok {
		return errors.New("expected table name")
	}
	if !p.tryKeyword("VALUES") {
		return errors.New("expected VALUES")
	}
	err := p.parseCommaList(func() error {
		return p.parseValueItem(&out.value)
	})
	if err != nil {
		return err
	}
	if !p.tryPunctuation(";") {
		return errors.New("expected ; ")
	}
	return nil
}
func (p *Parser) parseUpdate(out *StmtUpdate) error {
	var ok bool
	out.table, ok = p.tryName()
	if !ok {
		return fmt.Errorf("expected table name")
	}
	if !p.tryKeyword("SET") {
		return errors.New("expected SET")
	}
	for !p.tryKeyword("WHERE") {
		expr := NamedCell{}
		if len(out.value) > 0 && !p.tryPunctuation(",") {
			return errors.New("expected , ")
		}
		if err := p.parseEqual(&expr); err != nil {
			return err
		}
		out.value = append(out.value, expr)
	}
	if len(out.value) == 0 {
		return errors.New("expected assignment list")
	}
	p.pos -= len("WHERE")
	return p.parseWhere(&out.keys)
}
func (p *Parser) parseDelete(out *StmtDelete) error {
	var ok bool
	out.table, ok = p.tryName()
	if !ok {
		return fmt.Errorf("expected table name")
	}
	return p.parseWhere(&out.keys)
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
