package main

type Token struct {
	token string
	index int
}

func (t *Token) Token() string {
	return t.token
}
