package test

import (
	"fmt"
	"github.com/jonasfreyr/playground"
	"os"
	"testing"
)

func TestLexer(t *testing.T) {
	file := "lalli.txt"

	text, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	l, err := main.NewLexer()

	if err != nil {
		t.Fatal(err)
	}

	tokens := l.Tokenize(string(text))
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range tokens {
		for _, _ = range line {
			// TODO: fix this
			// fmt.Printf("<%s-%d,%d>", token.lexeme, token.location.col, token.location.line)
		}
		fmt.Println()
	}
}
