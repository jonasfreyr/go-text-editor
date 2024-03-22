package main

type Action int

const (
	INSERT Action = iota
	DELETE
	DELETE_LINE
)

type Transaction struct {
	location Location
	action   Action
	text     string
	amount   int
}
