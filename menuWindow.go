package main

import (
	gc "github.com/rthornton128/goncurses"
	"log"
)

type MenuWindow struct {
	stdscr *gc.Window
	menu   *gc.Menu
}

func NewMenuWindow(y, x, h, w int) (*MenuWindow, error) {
	stdscr, err := gc.NewWindow(h, w, y, x)
	if err != nil {
		log.Fatal(err)
	}
	err = stdscr.Keypad(true)
	if err != nil {
		log.Fatal(err)
	}

	menu, _ := gc.NewMenu([]*gc.MenuItem{})
	err = menu.SetWindow(stdscr)
	if err != nil {
		return nil, err
	}
	log.Println("This is the I:", h)
	dwin := stdscr.Derived(h-3, w-1, 3, 1)
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
	m.stdscr.Erase()

	// build the menu items
	menuItems := make([]*gc.MenuItem, len(items))
	for i, val := range items {
		log.Println(val)
		menuItems[i], _ = gc.NewItem(val, "")
		defer menuItems[i].Free()
	}

	_, x := m.stdscr.MaxYX()
	//m.stdscr.Resize(len(items)+4, x) // TODO: Needs fixing if it gets too big

	err := m.menu.SetItems(menuItems)
	if err != nil {
		return "", err
	}

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

	log.Println("Menu count:", m.menu.Count())

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
