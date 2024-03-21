package main

import (
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
)

type MiniWindow struct {
	x     int
	width int

	stdscr *gc.Window

	texts map[string]string
}

func (w *MiniWindow) draw(label string) {
	w.stdscr.Erase()
	// w.stdscr.Border(gc.ACS_VLINE, gc.ACS_VLINE, gc.ACS_HLINE, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS)
	w.stdscr.Print(label+":", w.texts[label])
	w.stdscr.Move(0, len(label)+1+w.x)
}

func (w *MiniWindow) moveX(delta int, label string) {
	w.x = utils.Max(utils.Min(w.x+delta, len(w.texts[label])), 0)
}

func (w *MiniWindow) run(clear bool, label string) string {
	if _, ok := w.texts[label]; !ok {
		w.texts[label] = ""
	}

	if clear {
		w.texts[label] = ""
	}
	w.x = len(w.texts[label])
	w.draw(label)

	for {
		k := w.stdscr.GetChar()

		switch k {
		case gc.KEY_ESC, gc.KEY_DOWN, gc.KEY_UP:
			return ""
		case gc.KEY_LEFT:
			w.moveX(-1, label)
		case gc.KEY_RIGHT:
			w.moveX(1, label)
		case gc.KEY_ENTER, gc.KEY_RETURN:
			return w.texts[label]
		case gc.KEY_TAB:
			w.texts[label] = w.texts[label][:w.x] + "\t" + w.texts[label][w.x:]
			w.moveX(1, label)
		case gc.KEY_END:
			w.x = len(w.texts[label])
		case gc.KEY_HOME:
			w.x = 0
		case gc.KEY_BACKSPACE:
			if w.x == 0 {
				continue
			}
			// e.lines[e.y] = e.lines[e.y][:e.x] + e.lines[e.y][e.x+num:]
			w.moveX(-1, label)
			w.texts[label] = w.texts[label][:w.x] + w.texts[label][w.x+1:]
		default:
			chr := gc.KeyString(k)
			if len(chr) > 1 {
				continue
			}

			w.texts[label] = w.texts[label][:w.x] + chr + w.texts[label][w.x:]
			w.x++
		}

		w.draw(label)
	}
}
