package main

type ActionType int

const (
	INSERT ActionType = iota
	DELETE
	DELETE_LINE
)

type Action struct {
	location Location
	action   ActionType
	text     string
	amount   int
}
