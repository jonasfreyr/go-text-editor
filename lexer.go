package main

import (
	"errors"
	"github.com/jonasfreyr/playground/utils"
	"io"
	"strings"
	"unicode"
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

func (t *Token) Length() int {
	if t.lexeme == "\n" {
		return 4
	}

	return len(t.lexeme)
}

func (t *Token) Token() string {
	return t.lexeme
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

func (l *Lexer) splitMultilineToken(token Token) []Token {
	newTokens := make([]Token, 0)
	newLexemes := strings.Split(token.lexeme, "\n")

	for _, lexeme := range newLexemes {
		newTokens = append(newTokens, l.newToken(lexeme, token.color, token.location))
	}
	return newTokens
}

func (l *Lexer) Reset() {
	l.eof = false
	l.ch = ""
	l.line = 0
	l.col = 0
}

func (l *Lexer) Tokenize(text string) [][]Token {
	l.Reset()
	tokens := make([][]Token, 0)
	tokens = append(tokens, make([]Token, 0))
	l.reader = strings.NewReader(text)
	l.read()
	lineIndex := 0
	for !l.eof {
		token := l.next()

		if token.lexeme == "\n" {
			tokens = append(tokens, make([]Token, 0))
			lineIndex++
			continue
		}

		if strings.Contains(token.lexeme, "\n") {
			newTokens := l.splitMultilineToken(token)

			for _, token := range newTokens {
				tokens[lineIndex] = append(tokens[lineIndex], token)

				lineIndex++
				tokens = append(tokens, make([]Token, 0))
			}
			continue
		}

		tokens[lineIndex] = append(tokens[lineIndex], token)
	}

	return tokens
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

	if errors.Is(err, io.EOF) {
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
	case "/": // TODO: don't hard code this comment thing
		l.read()
		if l.ch == "/" {
			// TODO: comment
			comment := "//"
			l.read()
			for l.ch != "\n" {
				comment += l.ch
				l.read()
			}
			return l.newToken(comment, l.config.Comment.Color, loc)
		} else if l.ch == "*" {
			str := "/*"
			l.read()
			for !l.eof {
				char := l.ch
				str += char
				l.read()

				if char == "*" && l.ch == "/" {
					str += l.ch
					l.read()
					break
				}
			}
			return l.newToken(str, l.config.Comment.Color, loc)
		} else {
			return l.newToken("/", l.config.Default.Color, loc)
		}
	case "\"":
		str := l.ch

		l.read()
		for !l.eof {
			char := l.ch
			str += char
			l.read()

			if char == "\"" {
				break
			}
		}

		return l.newToken(str, l.config.Strings.Color, loc)
	default:
		if unicode.IsLetter(rune(l.ch[0])) || l.ch == "_" {
			str := l.ch

			l.read()
			for unicode.IsLetter(rune(l.ch[0])) || unicode.IsNumber(rune(l.ch[0])) {
				str += l.ch
				l.read()
			}

			color := l.config.Default.Color
			if utils.Contains(l.config.Literals.Tokens, str) {
				color = l.config.Literals.Color
			} else if utils.Contains(l.config.BuiltIns.Tokens, str) {
				color = l.config.BuiltIns.Color
			} else if utils.Contains(l.config.Types.Tokens, str) {
				color = l.config.Types.Color
			} else if utils.Contains(l.config.Keywords.Tokens, str) {
				color = l.config.Keywords.Color
			}

			return l.newToken(str, color, loc)
		} else if unicode.IsNumber(rune(l.ch[0])) {
			number := l.ch
			l.read()
			for unicode.IsNumber(rune(l.ch[0])) {
				number += l.ch
				l.read()
			}
			return l.newToken(number, l.config.Digits.Color, loc)
		}
		ch := l.ch
		l.read()
		return l.newToken(ch, l.config.Default.Color, loc)
	}
}
