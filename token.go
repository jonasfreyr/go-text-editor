package main

type Token struct {
	token string
	index int
}

func (t *Token) Length() int {
	if t.token == "\n" {
		return 4
	}

	return len(t.token)
}

func (t *Token) Token() string {
	return t.token
}
