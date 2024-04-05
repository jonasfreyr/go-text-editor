package main

import (
	gc "github.com/rthornton128/goncurses"
)

type PopUpWindow struct {
	stdscr *gc.Window

	x, y int
}

func NewPopUpWindow(y, x, h, w int) (*PopUpWindow, error) {
	stdscr, err := gc.NewWindow(h, w, y, x)
	if err != nil {
		return nil, err
	}

	err = stdscr.Keypad(true)
	if err != nil {
		return nil, err
	}

	return &PopUpWindow{stdscr: stdscr, x: x, y: y}, nil
}

func (pw *PopUpWindow) pop(message string) {
	gc.Cursor(0)
	defer gc.Cursor(1)
	pw.stdscr.Erase()
	pw.stdscr.Resize(3, len(message)+2)
	pw.stdscr.MoveWindow(pw.y, pw.x-len(message)/2)
	y, x := pw.stdscr.MaxYX()
	pw.stdscr.Box(0, 0)
	pw.stdscr.MovePrint(1, 1, message)
	pw.stdscr.MoveAddChar(y, 0, gc.ACS_LTEE)
	pw.stdscr.HLine(y, 1, gc.ACS_HLINE, x-2)
	pw.stdscr.MoveAddChar(y, x-1, gc.ACS_RTEE)
	pw.stdscr.Refresh()

	pw.stdscr.GetChar()
}
