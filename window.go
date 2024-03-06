package main

import (
	gc "github.com/rthornton128/goncurses"
)

type Window interface {
	ColorOff(pair int16) error
	ColorOn(pair int16) error
	Clear() error
	Println(args ...interface{})
	Print(args ...interface{})
	Move(y int, x int)
	Refresh()
	Keypad(keypad bool) error
	MaxYX() (int, int)
	CursorYX() (int, int)
	GetChar() gc.Key
}

func NewDoubleBufferWindow() (*DoubleBufferWindow, error) {
	stdscr, err := gc.Init()

	if err != nil {
		return nil, err
	}

	h, w := stdscr.MaxYX()

	window1, err := gc.NewWindow(h, w, 0, 0)
	if err != nil {
		return nil, err
	}

	window2, err := gc.NewWindow(h, w, 0, 0)
	if err != nil {
		return nil, err
	}

	return &DoubleBufferWindow{
		currentWindow:    window1,
		backgroundWindow: window2,
	}, nil
}

type DoubleBufferWindow struct {
	currentWindow    *gc.Window
	backgroundWindow *gc.Window
}

func (w *DoubleBufferWindow) ColorOff(pair int16) error {
	return w.currentWindow.ColorOff(pair)
}

func (w *DoubleBufferWindow) ColorOn(pair int16) error {
	return w.currentWindow.ColorOn(pair)
}

func (w *DoubleBufferWindow) Clear() error {
	return w.currentWindow.Clear()
}

func (w *DoubleBufferWindow) Println(args ...interface{}) {
	w.currentWindow.Println(args...)
}

func (w *DoubleBufferWindow) Print(args ...interface{}) {
	w.currentWindow.Print(args...)
}

func (w *DoubleBufferWindow) Move(y int, x int) {
	w.currentWindow.Move(y, x)
}

func (w *DoubleBufferWindow) NRefresh() {
	w.currentWindow.Refresh()
}

func (w *DoubleBufferWindow) Refresh() {
	w.currentWindow.Refresh()
	w.currentWindow, w.backgroundWindow = w.backgroundWindow, w.currentWindow
}

func (w *DoubleBufferWindow) Keypad(keypad bool) error {
	err := w.currentWindow.Keypad(keypad)
	if err != nil {
		return err
	}

	return w.backgroundWindow.Keypad(keypad)
}

func (w *DoubleBufferWindow) MaxYX() (int, int) {
	return w.currentWindow.MaxYX()
}

func (w *DoubleBufferWindow) CursorYX() (int, int) {
	return w.currentWindow.CursorYX()
}

func (w *DoubleBufferWindow) GetChar() gc.Key {
	return w.currentWindow.GetChar()
}
