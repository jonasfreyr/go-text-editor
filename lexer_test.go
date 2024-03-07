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
	tokens := l.Tokenize(string(text))
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range tokens {
		for _, token := range line {
			fmt.Printf("<%s-%d,%d>", token.lexeme, token.location.col, token.location.line)
		}
		fmt.Println()
	}
}
