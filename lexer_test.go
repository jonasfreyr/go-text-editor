package main

import (
	"os"
	"testing"
)

func TestLexer(t *testing.T) {
	file := "lalli.txt"

	f, err := os.Open(file)

	if err != nil {
		t.Fatal(err)
	}

	var text []byte
	_, err = f.Read(text)

	if err != nil {
		t.Fatal(err)
	}

	l := NewLexer()
	tokens, err := l.Tokenize(string(text))
	if err != nil {
		t.Fatal(err)
	}
}
