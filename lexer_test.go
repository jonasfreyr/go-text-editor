package main

import (
	"fmt"
	"os"
	"testing"
)

func TestLexer(t *testing.T) {
	file := "lalli.txt"

	text, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	l := NewLexer()
	tokens, err := l.Tokenize(string(text))
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range tokens {
		fmt.Print("<" + token.lexeme + ">")
	}
}
