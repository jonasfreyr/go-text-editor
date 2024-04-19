package main

import (
	"github.com/jonasfreyr/gim/utils"
	gc "github.com/rthornton128/goncurses"
)

type MiniWindow struct {
	x     map[string]int
	width int

	stdscr *gc.Window

	texts map[string]string
}

func NewMiniWindow(y, x, h, w int) (*MiniWindow, error) {
	stdscr, err := gc.NewWindow(h, w, y, x)
	if err != nil {
		return nil, err
		//log.Fatal(err)
	}
	err = stdscr.Keypad(true)
	if err != nil {
		return nil, err
		//log.Fatal(err)
	}

	return &MiniWindow{width: w, stdscr: stdscr, texts: make(map[string]string), x: make(map[string]int)}, nil

}

func (w *MiniWindow) draw(label string) {
	w.stdscr.Erase()
	// w.stdscr.Border(gc.ACS_VLINE, gc.ACS_VLINE, gc.ACS_HLINE, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS)
	w.stdscr.Print(label+":", w.texts[label])
	w.stdscr.Move(0, len(label)+1+w.x[label])
}

func (w *MiniWindow) moveX(delta int, label string) {
	w.x[label] = utils.Max(utils.Min(w.x[label]+delta, len(w.texts[label])), 0)
}

func (w *MiniWindow) clear(label string) {
	w.texts[label] = ""
	w.x[label] = 0
}

func (w *MiniWindow) whileRun(clear bool, label string) string {
	if clear {
		w.clear(label)
	}

	w.draw(label)
	for {
		ch := w.stdscr.GetChar()

		switch ch {
		case gc.KEY_ESC:
			return ""
		case gc.KEY_ENTER, gc.KEY_RETURN:
			return w.texts[label]
		default:
			w.run(label, ch)
		}
	}
}

func (w *MiniWindow) run(label string, input gc.Key) string {
	if _, ok := w.texts[label]; !ok {
		w.clear(label)
	}

	//w.x = len(w.texts[label])
	//w.draw(label)

	//k := w.stdscr.GetChar()

	switch input {
	case gc.KEY_ESC:
		return ""
	case gc.KEY_LEFT:
		w.moveX(-1, label)
	case gc.KEY_RIGHT:
		w.moveX(1, label)
	case gc.KEY_ENTER, gc.KEY_RETURN:
		return w.texts[label]
	case gc.KEY_END:
		w.x[label] = len(w.texts[label])
	case gc.KEY_HOME:
		w.x[label] = 0
	case gc.KEY_BACKSPACE:
		if w.x[label] == 0 {
			return ""
		}
		// e.lines[e.y] = e.lines[e.y][:e.x] + e.lines[e.y][e.x+num:]
		w.moveX(-1, label)
		w.texts[label] = w.texts[label][:w.x[label]] + w.texts[label][w.x[label]+1:]
	default:
		chr := gc.KeyString(input)
		if len(chr) > 1 {
			return ""
		}

		w.texts[label] = w.texts[label][:w.x[label]] + chr + w.texts[label][w.x[label]:]
		w.moveX(1, label)
	}

	w.draw(label)
	return ""
}
