package main

import (
	"errors"
	"io"
	"strings"
)

type Location struct {
	line int
	col  int
}

type Token struct {
	color    [3]int
	lexeme   string
	location Location
}

type Lexer struct {
	config *JSONConfig
	reader *strings.Reader

	ch  string
	eof bool

	line, col int
}

func NewLexer() *Lexer {
	lexer := Lexer{}

	config, err := ReadConfig("highlighting.json")
	if err != nil {
		return nil
	}

	lexer.config = config

	return &lexer
}

func (l *Lexer) Tokenize(text string) ([]Token, error) {
	tokens := make([]Token, 0)
	l.reader = strings.NewReader(text)
	l.read()
	for token := l.next(); token.lexeme != ""; token = l.next() {
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (l *Lexer) read() {
	if l.eof {
		l.ch = ""
	}

	if l.ch == "\n" {
		l.line++
		l.col = 1
	} else {
		l.col++
	}

	var newChar = make([]byte, 1)
	_, err := l.reader.Read(newChar)

	if errors.Is(io.EOF, err) {
		l.line++
		l.col = 1
		l.ch = "\n"
		l.eof = true
	} else if err != nil {
		l.ch = ""
		l.eof = true
	} else {
		l.ch = string(newChar)
	}
}

func (l *Lexer) newToken(ch string, color [3]int, loc Location) Token {
	return Token{
		color:    color,
		lexeme:   ch,
		location: loc,
	}
}

func (l *Lexer) next() Token {
	loc := Location{
		line: l.line,
		col:  l.col,
	}

	switch l.ch {
	case " ", "\t", "\n", "+", "-", "=", "*":
		return l.newToken(l.ch, [3]int{}, loc)
	case "/": // TODO: don't hard code this comment thing
		l.read()
		if l.ch == "/" {
			// TODO: comment
			comment := "//"
			l.next()
			for l.ch != "\n" {
				comment += l.ch
				l.next()
			}
			return l.newToken(comment, l.config.Comment.Color, loc)
		} else {
			return l.newToken("/", [3]int{}, loc)
		}

	default:
		return l.newToken(l.ch, [3]int{}, loc)
	}
}
