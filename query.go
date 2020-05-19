//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrorOptionalMissing = errors.New("optional element missing")
)

type Query struct {
	left     *Query
	optional bool
	key      string
	filters  []Filter
}

func (q *Query) String() string {
	var str, opt string
	if q.optional {
		opt = "?"
	}
	if q.left != nil {
		str = fmt.Sprintf("%s.%s%q", q.left.String(), opt, q.key)
	} else {
		str = fmt.Sprintf("%s%q", opt, q.key)
	}
	for _, f := range q.filters {
		str += fmt.Sprintf("[%s]", f.String())
	}
	return str
}

type Filter interface {
	String() string
	Eval(index int, v interface{}) (bool, error)
}

func (q *Query) Eval(v interface{}) (interface{}, error) {
	var err error

	if q.left != nil {
		v, err = q.left.Eval(v)
		if err != nil {
			return nil, err
		}
	}

	// Select by key.
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("jsonq: query '%s' can't index %T", q, v)
	}
	child, ok := m[q.key]
	if !ok {
		if q.optional {
			return nil, ErrorOptionalMissing
		}
		return nil, fmt.Errorf("jsonq: element '%s' not found", q)
	}
	v = child

	if len(q.filters) == 0 {
		return v, nil
	}

	var result []interface{}
	arr, ok := v.([]interface{})
	if ok {
		result = arr
	} else {
		result = []interface{}{v}
	}

	for _, filter := range q.filters {
		var filtered []interface{}

		for idx, item := range result {
			ok, err := filter.Eval(idx, item)
			if err != nil {
				return nil, err
			}
			if ok {
				filtered = append(filtered, item)
			}
		}

		result = filtered
	}
	return result, nil
}

func Parse(q string) (*Query, error) {
	return parseQuery(NewLexer(q))
}

func parseQuery(lexer *Lexer) (*Query, error) {
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	var optional bool
	if t.Type == tQuestionMark {
		optional = true
		t, err = lexer.Get()
		if err != nil {
			return nil, err
		}
	}

	if t.Type != tString {
		return nil, lexer.SyntaxError()
	}
	q := &Query{
		optional: optional,
		key:      t.StrVal,
	}
	for {
		t, err = lexer.Get()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		if t.Type != tDot {
			lexer.Unget(t)
			break
		}
		t, err = lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type != tString {
			return nil, lexer.SyntaxError()
		}
		q = &Query{
			left: q,
			key:  t.StrVal,
		}
	}

	// Filters.
	for {
		t, err = lexer.Get()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if t.Type != tLBracket {
			return nil, lexer.SyntaxError()
		}
		filter, err := parseLogical(lexer)
		if err != nil {
			return nil, err
		}
		q.filters = append(q.filters, filter)
	}

	return q, nil
}

func parseLogical(lexer *Lexer) (Filter, error) {
	left, err := parseComparative(lexer)
	if err != nil {
		return nil, err
	}
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tRBracket:
		return left, nil

	case tAnd, tOr:
		right, err := parseComparative(lexer)
		if err != nil {
			return nil, err
		}
		return &Logical{
			Left:  left,
			Op:    t.Type,
			Right: right,
		}, nil

	default:
		return nil, lexer.SyntaxError()
	}
}

func parseComparative(lexer *Lexer) (Filter, error) {
	left, err := parseAtom(lexer)
	if err != nil {
		return nil, err
	}
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tEq, tNeq:
		right, err := parseAtom(lexer)
		if err != nil {
			return nil, err
		}
		return &Comparative{
			Left:  left,
			Op:    t.Type,
			Right: right,
		}, nil

	default:
		lexer.Unget(t)
		return &Comparative{
			Left: left,
			Op:   left.Type,
		}, nil
	}
}

func parseAtom(lexer *Lexer) (*Atom, error) {
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tString:
		return &Atom{
			Type:   t.Type,
			StrVal: t.StrVal,
		}, nil

	case tInt:
		return &Atom{
			Type: t.Type,
			Int:  t.Int,
		}, nil

	default:
		return nil, lexer.SyntaxError()
	}
}

type Select struct {
	Left *Select
	Key  string
}

func (ast *Select) Eval(idx int, v interface{}) (bool, error) {
	return false, fmt.Errorf("Select.Eval not implemented yet")
}

type Logical struct {
	Left  Filter
	Op    TokenType
	Right Filter
}

func (ast *Logical) String() string {
	return fmt.Sprintf("%s%s%s", ast.Left, ast.Op, ast.Right)
}

func (ast *Logical) Eval(idx int, v interface{}) (bool, error) {
	lVal, err := ast.Left.Eval(idx, v)
	if err != nil {
		return false, err
	}
	rVal, err := ast.Right.Eval(idx, v)
	if err != nil {
		return false, err
	}
	switch ast.Op {
	case tAnd:
		return lVal && rVal, nil

	case tOr:
		return lVal || rVal, nil

	default:
		return false, fmt.Errorf("invalid logical operation %s", ast.Op)
	}
}

type Comparative struct {
	Left  *Atom
	Op    TokenType
	Right *Atom
}

func (ast *Comparative) String() string {
	return fmt.Sprintf("%s%s%s", ast.Left, ast.Op, ast.Right)
}

func (ast *Comparative) Eval(idx int, v interface{}) (bool, error) {
	fmt.Printf("Comparative.Eval: %T\n", v)
	switch ast.Op {
	case tEq:
		switch ast.Right.Type {
		case tString:
			field, err := ast.Left.GetString()
			if err != nil {
				return false, err
			}
			val, err := GetString(v, field)
			if err != nil {
				return false, err
			}
			return val == ast.Right.StrVal, nil

		default:
			return false, fmt.Errorf("== not implemented for %s",
				ast.Right.Type)
		}

	case tInt:
		return idx == ast.Left.Int, nil

	default:
		return false, fmt.Errorf("Comparative.Eval %s not implemented yet",
			ast.Op)
	}
}

type Atom struct {
	Type   TokenType
	StrVal string
	Int    int
}

func (a *Atom) String() string {
	switch a.Type {
	case tString:
		return fmt.Sprintf("%q", a.StrVal)

	case tInt:
		return fmt.Sprintf("%v", a.Int)

	default:
		return fmt.Sprintf("{Atom %d}", a.Type)
	}
}

func (a *Atom) GetString() (string, error) {
	switch a.Type {
	case tString:
		return a.StrVal, nil

	default:
		return "", fmt.Errorf("not string value %s", a.Type)
	}
}
