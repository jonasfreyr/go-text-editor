package main

import (
	gc "github.com/rthornton128/goncurses"
	"log"
	"strings"
)

type MenuWindow struct {
	stdscr    *gc.Window
	subWindow *gc.Window

	selected   int
	itemOffSet int

	mark string
}

type MenuItem struct {
	label string
	color [3]int
}

func NewMenuWindow(y, x, h, w int) (*MenuWindow, error) {
	stdscr, err := gc.NewWindow(h, w, y, x)
	if err != nil {
		return nil, err
	}

	err = stdscr.Keypad(true)
	if err != nil {
		return nil, err
	}

	log.Println("This is the I:", h)
	dwin := stdscr.Derived(h-4, w-2, 3, 1)

	mw := &MenuWindow{
		stdscr:    stdscr,
		subWindow: dwin,
		mark:      " -> ", // TODO: Maybe put in config?
	}
	return mw, nil
}

func (m *MenuWindow) drawBorderAndTitle(title string) {
	m.stdscr.Erase()
	_, x := m.stdscr.MaxYX()
	m.stdscr.Box(0, 0)
	m.stdscr.MovePrint(1, (x/2)-(len(title)/2), title)
	m.stdscr.MoveAddChar(2, 0, gc.ACS_LTEE)
	m.stdscr.HLine(2, 1, gc.ACS_HLINE, x-2)
	m.stdscr.MoveAddChar(2, x-1, gc.ACS_RTEE)
	m.stdscr.Refresh()
}

func (m *MenuWindow) drawMenu(items []MenuItem) {
	m.subWindow.Erase()
	m.subWindow.Move(0, 0)
	y, x := m.subWindow.MaxYX()

	if m.selected >= y+m.itemOffSet {
		m.itemOffSet = m.selected - y + 1
	} else if m.selected < m.itemOffSet {
		m.itemOffSet = m.selected
	}

	for i, item := range items[m.itemOffSet:] {
		if i >= y {
			break
		}

		itemLabel := item.label

		if len(itemLabel)+len(m.mark) >= x {
			itemLabel = itemLabel[:x]
		}

		if m.selected == i+m.itemOffSet {
			m.subWindow.Print(m.mark)
			m.subWindow.AttrOn(gc.A_REVERSE)
			EnableColor(m.subWindow, item.color)
			m.subWindow.Println(itemLabel)
			DisableColor(m.subWindow, item.color)
			m.subWindow.AttrOff(gc.A_REVERSE)
		} else {
			EnableColor(m.subWindow, item.color)
			m.subWindow.Println(strings.Repeat(" ", len(m.mark)) + itemLabel)
			DisableColor(m.subWindow, item.color)
		}
	}
	m.subWindow.Refresh()
}

func (m *MenuWindow) run(items []MenuItem, title string) (string, error) {
	gc.Cursor(0)
	defer gc.Cursor(1)

	m.drawBorderAndTitle(title)
	m.selected = 0
	m.itemOffSet = 0
	for {
		m.drawMenu(items)
		ch := m.stdscr.GetChar()

		switch ch {
		case gc.KEY_ENTER, gc.KEY_RETURN:
			return items[m.selected].label, nil
		case gc.KEY_ESC:
			return "", nil
		case gc.KEY_DOWN:
			m.selected++

			if m.selected > len(items)-1 {
				m.selected = 0
			}
		case gc.KEY_UP:
			m.selected--
			if m.selected < 0 {
				m.selected = len(items) - 1
			}
		}
	}
}
