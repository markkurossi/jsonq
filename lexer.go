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
	tAND TokenType = iota
	tOR
	tEQ
	tNEQ
	tSymbol
	tString
	tInt
)

var tokens = map[TokenType]string{
	tAND:    "&&",
	tOR:     "||",
	tEQ:     "==",
	tNEQ:    "!=",
	tSymbol: "symbol",
	tString: "string",
	tInt:    "int",
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
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		in:    bufio.NewReader(strings.NewReader(input)),
	}
}

func (l *Lexer) Get() (*Token, error) {
	r, _, err := l.ReadRune()
	if err != nil {
		return nil, err
	}
	switch r {
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
			Type: tEQ,
		}, nil

	case '"':
		var val []rune
		for {
			r, _, err = l.ReadRune()
			if err != nil {
				return nil, err
			}
			if r == '"' {
				break
			}
			// XXX escapes
			val = append(val, r)
		}
		return &Token{
			Type:   tString,
			StrVal: string(val),
		}, nil

	default:
		if unicode.IsLetter(r) {
			symbol := []rune{r}
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
				symbol = append(symbol, r)
			}
			return &Token{
				Type:   tString,
				StrVal: string(symbol),
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
	return fmt.Errorf("syntax error: '%s'...",
		string([]byte(l.input)[:l.pos]))
}
