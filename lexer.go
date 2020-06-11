//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type tokenType int

const (
	tDot tokenType = iota
	tLBracket
	tRBracket
	tQuestionMark
	tAnd
	tOr
	tEq
	tNeq
	tLt
	tLe
	tGt
	tGe
	tString
	tInt
)

var tokens = map[tokenType]string{
	tDot:          ".",
	tLBracket:     "[",
	tRBracket:     "]",
	tQuestionMark: "?",
	tAnd:          "&&",
	tOr:           "||",
	tEq:           "==",
	tNeq:          "!=",
	tLt:           "<",
	tLe:           "<=",
	tGt:           ">",
	tGe:           ">=",
	tString:       "string",
	tInt:          "int",
}

func (tt tokenType) String() string {
	name, ok := tokens[tt]
	if ok {
		return name
	}
	return fmt.Sprintf("{tokenType %d}", tt)
}

type token struct {
	Type   tokenType
	StrVal string
	Int    int
}

type lexer struct {
	input    string
	in       *bufio.Reader
	pos      int
	lastSize int
	unget    *token
}

func newLexer(input string) *lexer {
	return &lexer{
		input: input,
		in:    bufio.NewReader(strings.NewReader(input)),
	}
}

func (l *lexer) Get() (*token, error) {
	if l.unget != nil {
		ret := l.unget
		l.unget = nil
		return ret, nil
	}
	r, _, err := l.ReadRune()
	if err != nil {
		return nil, err
	}
	switch r {
	case '.':
		return &token{
			Type: tDot,
		}, nil

	case '[':
		return &token{
			Type: tLBracket,
		}, nil

	case ']':
		return &token{
			Type: tRBracket,
		}, nil

	case '?':
		return &token{
			Type: tQuestionMark,
		}, nil

	case '=':
		r, _, err = l.ReadRune()
		if err != nil {
			return nil, err
		}
		if r != '=' {
			l.UnreadRune()
			return nil, l.SyntaxError()
		}
		return &token{
			Type: tEq,
		}, nil

	case '!':
		r, _, err := l.ReadRune()
		if err != nil {
			return nil, err
		}
		if r != '=' {
			l.UnreadRune()
			return nil, l.SyntaxError()
		}
		return &token{
			Type: tNeq,
		}, nil

	case '<':
		r, _, err := l.ReadRune()
		if err != nil {
			return nil, err
		}
		if r == '=' {
			return &token{
				Type: tLe,
			}, nil
		}
		l.UnreadRune()
		return &token{
			Type: tLt,
		}, nil

	case '>':
		r, _, err := l.ReadRune()
		if err != nil {
			return nil, err
		}
		if r == '=' {
			return &token{
				Type: tGe,
			}, nil
		}
		l.UnreadRune()
		return &token{
			Type: tGt,
		}, nil

	case '"':
		var str []rune
		for {
			r, _, err = l.ReadRune()
			if err != nil {
				return nil, err
			}
			if r == '"' {
				break
			}
			// XXX escapes
			str = append(str, r)
		}
		return &token{
			Type:   tString,
			StrVal: string(str),
		}, nil

	default:
		if unicode.IsLetter(r) {
			str := []rune{r}
			for {
				r, _, err = l.ReadRune()
				if err != nil {
					if err != io.EOF {
						return nil, err
					}
					break
				}
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					l.UnreadRune()
					break
				}
				str = append(str, r)
			}
			return &token{
				Type:   tString,
				StrVal: string(str),
			}, nil
		}
		if unicode.IsDigit(r) {
			number := []rune{r}
			for {
				r, _, err = l.ReadRune()
				if err != nil {
					if err != io.EOF {
						return nil, err
					}
					break
				}
				if !unicode.IsDigit(r) {
					l.UnreadRune()
					break
				}
				number = append(number, r)
			}
			ival, err := strconv.Atoi(string(number))
			if err != nil {
				return nil, err
			}
			return &token{
				Type: tInt,
				Int:  ival,
			}, nil
		}

		l.UnreadRune()
		return nil, l.SyntaxError()
	}
}

func (l *lexer) Unget(t *token) {
	l.unget = t
}

func (l *lexer) ReadRune() (rune, int, error) {
	r, s, err := l.in.ReadRune()
	l.lastSize = s
	l.pos += s
	return r, s, err
}

func (l *lexer) UnreadRune() error {
	err := l.in.UnreadRune()
	if err != nil {
		return err
	}
	l.pos -= l.lastSize
	return nil
}

func (l *lexer) SyntaxError() error {
	if l.pos == 0 {
		return fmt.Errorf("syntax error at the beginning of query '%s'",
			l.input)
	}
	return fmt.Errorf("syntax error: '%s', looking at '%s'",
		string([]byte(l.input)[:l.pos]),
		string([]byte(l.input)[l.pos:]))
}
