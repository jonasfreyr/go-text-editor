package main

import (
	gc "github.com/rthornton128/goncurses"
	"log"
)

type MenuWindow struct {
	stdscr *gc.Window
	menu   *gc.Menu
}

func NewMenuWindow(stdscr *gc.Window) (*MenuWindow, error) {
	menu, _ := gc.NewMenu([]*gc.MenuItem{})
	err := menu.SetWindow(stdscr)
	if err != nil {
		return nil, err
	}

	dwin := stdscr.Derived(6, 38, 3, 1) // TODO: figure out what this actually does
	err = menu.SubWindow(dwin)
	if err != nil {
		return nil, err
	}
	err = menu.Mark(" -> ")
	if err != nil {
		return nil, err
	}

	mw := &MenuWindow{
		stdscr: stdscr,
		menu:   menu,
	}
	return mw, nil
}

func (m *MenuWindow) Free() {
	err := m.menu.Free()
	if err != nil {
		log.Println("error freeing menu:", err)
	}
}

func (m *MenuWindow) run(items []string, title string) (string, error) {
	gc.Cursor(0)
	defer gc.Cursor(1)

	// build the menu items
	menuItems := make([]*gc.MenuItem, len(items))
	for i, val := range items {
		menuItems[i], _ = gc.NewItem(val, "")
		defer menuItems[i].Free()
	}

	err := m.menu.SetItems(menuItems)
	if err != nil {
		return "", err
	}

	// Print centered menu title
	m.stdscr.Erase()
	_, x := m.stdscr.MaxYX()
	m.stdscr.Box(0, 0)
	m.stdscr.MovePrint(1, (x/2)-(len(title)/2), title)
	m.stdscr.MoveAddChar(2, 0, gc.ACS_LTEE)
	m.stdscr.HLine(2, 1, gc.ACS_HLINE, x-2)
	m.stdscr.MoveAddChar(2, x-1, gc.ACS_RTEE)

	err = m.menu.Post()
	if err != nil {
		return "", err
	}

	defer m.menu.UnPost()
	m.stdscr.Refresh()

	for {
		gc.Update()
		ch := m.stdscr.GetChar()

		switch ch {
		case gc.KEY_ENTER, gc.KEY_RETURN:
			current := m.menu.Current(nil)
			return current.Name(), nil
		case gc.KEY_ESC:
			return "", nil
		case gc.KEY_DOWN:
			m.menu.Driver(gc.REQ_DOWN)
		case gc.KEY_UP:
			m.menu.Driver(gc.REQ_UP)
		}
	}
}
