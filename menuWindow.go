package main

import (
	gc "github.com/rthornton128/goncurses"
	"strings"
)

type MenuWindow struct {
	stdscr    *gc.Window
	subWindow *gc.Window

	selected   int
	itemOffSet int

	mark string

	items []MenuItem
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

	//log.Println("This is the I:", h)
	dwin := stdscr.Derived(h-4, w-2, 3, 1)

	mw := &MenuWindow{
		stdscr:    stdscr,
		subWindow: dwin,
		mark:      " -> ", // TODO: Maybe put in config?
	}
	return mw, nil
}

func (m *MenuWindow) drawBorderAndTitle(title string) {
	_, x := m.stdscr.MaxYX()
	m.stdscr.Box(0, 0)
	// (x/2)-(len(title)/2)
	m.stdscr.MovePrint(1, 2, title)
	m.stdscr.MoveAddChar(2, 0, gc.ACS_LTEE)
	m.stdscr.HLine(2, 1, gc.ACS_HLINE, x-2)
	m.stdscr.MoveAddChar(2, x-1, gc.ACS_RTEE)
}

func (m *MenuWindow) drawMenu() {
	m.subWindow.Move(0, 0)
	y, x := m.subWindow.MaxYX()

	if m.selected >= y+m.itemOffSet {
		m.itemOffSet = m.selected - y + 1
	} else if m.selected < m.itemOffSet {
		m.itemOffSet = m.selected
	}

	for i, item := range m.items[m.itemOffSet:] {
		if i >= y {
			break
		}

		itemLabel := item.label

		if len(itemLabel)+len(m.mark) >= x-2 {
			itemLabel = itemLabel[len(itemLabel)+len(m.mark)+2-x:]
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
			m.subWindow.Println(strings.Repeat(" ", len(m.mark)) + strings.ReplaceAll(strings.ReplaceAll(itemLabel, "\r", ""), "\n", ""))
			DisableColor(m.subWindow, item.color)
		}
	}
}

func (m *MenuWindow) draw(title string) {
	m.stdscr.Erase()
	m.drawBorderAndTitle(title)
	m.drawMenu()
	m.stdscr.Refresh()
}

func (m *MenuWindow) setItems(items []MenuItem) {
	m.items = items
	m.selected = 0
	m.itemOffSet = 0
}

func (m *MenuWindow) run(input gc.Key) string {
	//gc.Cursor(0)
	//defer gc.Cursor(1)

	switch input {
	case gc.KEY_ENTER, gc.KEY_RETURN:
		return m.items[m.selected].label
	case gc.KEY_ESC:
		return ""
	case gc.KEY_DOWN:
		m.selected++

		if m.selected > len(m.items)-1 {
			m.selected = 0
		}
	case gc.KEY_UP:
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.items) - 1
		}
	}
	return ""
}
