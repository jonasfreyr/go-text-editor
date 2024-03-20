package main

import (
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
)

type MiniWindow struct {
	text  string
	x     int
	width int

	stdscr *gc.Window
}

func (w *MiniWindow) draw() {
	w.stdscr.Erase()
	// w.stdscr.Border(gc.ACS_VLINE, gc.ACS_VLINE, gc.ACS_HLINE, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS)
	w.stdscr.Print(w.text)
	w.stdscr.Move(0, w.x)
}

func (w *MiniWindow) moveX(delta int) {
	w.x = utils.Max(utils.Min(w.x+delta, len(w.text)), 0)
}

func (w *MiniWindow) run(clear bool) string {
	if clear {
		w.text = ""
		w.x = 0
	}
	w.draw()

	for {
		key := w.stdscr.GetChar()

		switch key {
		case gc.KEY_ESC:
			return ""
		case gc.KEY_LEFT:
			w.moveX(-1)
		case gc.KEY_RIGHT:
			w.moveX(1)
		case gc.KEY_ENTER, gc.KEY_RETURN:
			return w.text
		case gc.KEY_TAB:
			w.text = w.text[:w.x] + "\t" + w.text[w.x:]
			w.moveX(1)
		case gc.KEY_END:
			w.x = len(w.text)
		case gc.KEY_HOME:
			w.x = 0
		case gc.KEY_BACKSPACE:
			if w.x == 0 {
				continue
			}
			// e.lines[e.y] = e.lines[e.y][:e.x] + e.lines[e.y][e.x+num:]
			w.moveX(-1)
			w.text = w.text[:w.x] + w.text[w.x+1:]
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			w.text = w.text[:w.x] + chr + w.text[w.x:]
			w.x++
		}

		w.draw()
	}
}
