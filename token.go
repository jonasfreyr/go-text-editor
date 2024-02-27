package main

type TextToken struct {
	token string
	index int
}

func (t *TextToken) Length() int {
	if t.token == "\n" {
		return 4
	}

	return len(t.token)
}

func (t *TextToken) Token() string {
	return t.token
}
