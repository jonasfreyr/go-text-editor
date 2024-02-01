package main

import (
	"github.com/rthornton128/goncurses"
	"log"
)

func main() {
	// Initialize goncurses. It's essential End() is called to ensure the
	// terminal isn't altered after the program ends
	stdscr, err := goncurses.Init()

	if err != nil {
		log.Fatal("init", err)
	}
	defer goncurses.End()

	// stdscr.Print("Hello, World!")
	// stdscr.Move(1, 0)
	goncurses.Echo(false)

	maxY, _ := stdscr.MaxYX()

	currentText := make([]string, maxY)

	for {
		key := stdscr.GetChar()
		y, x := stdscr.CursorYX()

		switch key {
		case goncurses.KEY_ESC:
			return
		case goncurses.KEY_ENTER, goncurses.KEY_RETURN:
			currentText[y] += "\n"
			stdscr.Println("")
		case goncurses.KEY_TAB:

		case 127: // Backspace?
			if x == 0 {
				if y == 0 {
					continue
				}
				y -= 1
				x = len(currentText[y]) - 1
			} else {
				x--
			}
			stdscr.Move(y, x)
			err := stdscr.DelChar()
			currentText[y] = currentText[y][:x] + currentText[y][x+1:]
			if err != nil {
				panic(err)
			}

		default:
			chr := goncurses.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			stdscr.AddChar(goncurses.Char(key))
			currentText[y] += chr
		}

		stdscr.Refresh()
	}
}
