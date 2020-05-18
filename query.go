//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"fmt"
	"io"
)

type Query struct {
	filter Filter
}

func (f *Query) Eval(v []interface{}) ([]interface{}, error) {
	var result []interface{}
	for idx, item := range v {
		ok, err := f.filter.Eval(idx, item)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, item)
		}
	}
	return result, nil
}

type Filter interface {
	Eval(index int, v interface{}) (bool, error)
}

func Parse(q string) (*Query, error) {
	filter, err := parseLogical(NewLexer(q))
	if err != nil {
		return nil, err
	}
	return &Query{
		filter: filter,
	}, nil
}

func parseLogical(lexer *Lexer) (Filter, error) {
	left, err := parseComparative(lexer)
	if err != nil {
		return nil, err
	}
	t, err := lexer.Get()
	if err != nil {
		if err == io.EOF {
			return left, nil
		}
		return nil, err
	}
	switch t.Type {
	case tAND, tOR:
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
		if err == io.EOF {
			return &Comparative{
				Left: left,
				Op:   left.Type,
			}, nil
		}
		return nil, err
	}
	switch t.Type {
	case tEQ, tNEQ:
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
		return nil, lexer.SyntaxError()
	}
}

func parseAtom(lexer *Lexer) (*Atom, error) {
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tSymbol, tString:
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

type Logical struct {
	Left  Filter
	Op    TokenType
	Right Filter
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
	case tAND:
		return lVal && rVal, nil

	case tOR:
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

func (ast *Comparative) Eval(idx int, v interface{}) (bool, error) {
	fmt.Printf("Comparative.Eval: %T\n", v)
	switch ast.Op {
	case tEQ:
		switch ast.Right.Type {
		case tSymbol, tString:
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

func (a *Atom) GetString() (string, error) {
	switch a.Type {
	case tSymbol, tString:
		return a.StrVal, nil

	default:
		return "", fmt.Errorf("not string value %s", a.Type)
	}
}
