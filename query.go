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
	"strings"
)

var (
	// ErrorOptionalMissing is returned when an optional element is
	// missing from the JSON object.
	ErrorOptionalMissing = errors.New("optional element missing")
)

type query struct {
	left     *query
	optional bool
	key      string
	filters  []filter
}

func (q *query) String() string {
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

type filter interface {
	String() string
	Eval(index int, v interface{}) (bool, error)
}

func (q *query) Eval(v interface{}) (interface{}, error) {
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

func parse(q string) (*query, error) {
	return parseQuery(newLexer(q))
}

func parseQuery(lexer *lexer) (*query, error) {
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
	q := &query{
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
		q = &query{
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

func parseLogical(lexer *lexer) (filter, error) {
	left, err := parseComparative(lexer)
	if err != nil {
		return nil, err
	}
	for {
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
			left = &logical{
				Left:  left,
				Op:    t.Type,
				Right: right,
			}

		default:
			return nil, lexer.SyntaxError()
		}
	}
}

func parseComparative(lexer *lexer) (filter, error) {
	left, err := parseAtom(lexer)
	if err != nil {
		return nil, err
	}
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tEq, tNeq, tLt, tLe, tGt, tGe:
		right, err := parseAtom(lexer)
		if err != nil {
			return nil, err
		}
		return &comparative{
			Left:  left,
			Op:    t.Type,
			Right: right,
		}, nil

	default:
		lexer.Unget(t)
		return &comparative{
			Left: left,
			Op:   left.Type,
		}, nil
	}
}

func parseAtom(lexer *lexer) (*atom, error) {
	t, err := lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case tString:
		return &atom{
			Type:   t.Type,
			StrVal: t.StrVal,
		}, nil

	case tInt:
		return &atom{
			Type:   t.Type,
			IntVal: t.Int,
		}, nil

	default:
		return nil, lexer.SyntaxError()
	}
}

type logical struct {
	Left  filter
	Op    tokenType
	Right filter
}

func (ast *logical) String() string {
	return fmt.Sprintf("%s%s%s", ast.Left, ast.Op, ast.Right)
}

func (ast *logical) Eval(idx int, v interface{}) (bool, error) {
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

type comparative struct {
	Left  *atom
	Op    tokenType
	Right *atom
}

func (ast *comparative) String() string {
	return fmt.Sprintf("%s%s%s", ast.Left, ast.Op, ast.Right)
}

func (ast *comparative) Eval(idx int, v interface{}) (bool, error) {
	switch ast.Op {
	case tEq:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return val == ast.Right.StrVal, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val == ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tNeq:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return val != ast.Right.StrVal, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val != ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tLt:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return strings.Compare(val, ast.Right.StrVal) < 0, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val < ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tLe:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return strings.Compare(val, ast.Right.StrVal) <= 0, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val <= ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tGt:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return strings.Compare(val, ast.Right.StrVal) > 0, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val > ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tGe:
		switch ast.Right.Type {
		case tString:
			val, err := ast.Left.GetStringField(v)
			if err != nil {
				return false, err
			}
			return strings.Compare(val, ast.Right.StrVal) >= 0, nil

		case tInt:
			val, err := ast.Left.GetIntField(v)
			if err != nil {
				return false, err
			}
			return val >= ast.Right.IntVal, nil

		default:
			return false, fmt.Errorf("%s not implemented for %s", ast.Op,
				ast.Right.Type)
		}

	case tInt:
		return idx == ast.Left.IntVal, nil

	default:
		return false, fmt.Errorf("Comparative.Eval %s not implemented yet",
			ast.Op)
	}
}

type atom struct {
	Type   tokenType
	StrVal string
	IntVal int
}

func (a *atom) String() string {
	switch a.Type {
	case tString:
		return fmt.Sprintf("%q", a.StrVal)

	case tInt:
		return fmt.Sprintf("%v", a.IntVal)

	default:
		return fmt.Sprintf("{atom %d}", a.Type)
	}
}

func (a *atom) GetString() (string, error) {
	switch a.Type {
	case tString:
		return a.StrVal, nil

	default:
		return "", fmt.Errorf("not string value %s", a.Type)
	}
}

func (a *atom) GetStringField(value interface{}) (string, error) {
	field, err := a.GetString()
	if err != nil {
		return "", err
	}
	return GetString(value, field)
}

func (a *atom) GetIntField(value interface{}) (int, error) {
	field, err := a.GetString()
	if err != nil {
		return 0, err
	}
	return GetInt(value, field)
}
