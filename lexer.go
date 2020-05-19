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

type TokenType int

const (
	tDot TokenType = iota
	tLBracket
	tRBracket
	tQuestionMark
	tAnd
	tOr
	tEq
	tNeq
	tString
	tInt
)

var tokens = map[TokenType]string{
	tDot:          ".",
	tLBracket:     "[",
	tRBracket:     "]",
	tQuestionMark: "?",
	tAnd:          "&&",
	tOr:           "||",
	tEq:           "==",
	tNeq:          "!=",
	tString:       "string",
	tInt:          "int",
}

func (tt TokenType) String() string {
	name, ok := tokens[tt]
	if ok {
		return name
	}
	return fmt.Sprintf("{TokenType %d}", tt)
}

type Token struct {
	Type   TokenType
	StrVal string
	Int    int
}

type Lexer struct {
	input    string
	in       *bufio.Reader
	pos      int
	lastSize int
	unget    *Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		in:    bufio.NewReader(strings.NewReader(input)),
	}
}

func (l *Lexer) Get() (*Token, error) {
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
		return &Token{
			Type: tDot,
		}, nil

	case '[':
		return &Token{
			Type: tLBracket,
		}, nil

	case ']':
		return &Token{
			Type: tRBracket,
		}, nil

	case '?':
		return &Token{
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
		return &Token{
			Type: tEq,
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
		return &Token{
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
			return &Token{
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
			return &Token{
				Type: tInt,
				Int:  ival,
			}, nil
		}

		l.UnreadRune()
		return nil, l.SyntaxError()
	}
}

func (l *Lexer) Unget(t *Token) {
	l.unget = t
}

func (l *Lexer) ReadRune() (rune, int, error) {
	r, s, err := l.in.ReadRune()
	l.lastSize = s
	l.pos += s
	return r, s, err
}

func (l *Lexer) UnreadRune() error {
	err := l.in.UnreadRune()
	if err != nil {
		return err
	}
	l.pos -= l.lastSize
	return nil
}

func (l *Lexer) SyntaxError() error {
	if l.pos == 0 {
		return fmt.Errorf("syntax error at the beginning of query '%s'",
			l.input)
	}
	return fmt.Errorf("syntax error: '%s', looking at '%s'",
		string([]byte(l.input)[:l.pos]),
		string([]byte(l.input)[l.pos:]))
}
