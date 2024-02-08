package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	gc "github.com/rthornton128/goncurses"
)

type Editor struct {
	text   []string
	stdscr *gc.Window

	maxX, maxY int

	x, y int

	currLengthIndex int

	printLineStartIndex int
	printLinesIndex     int
}

func (e *Editor) draw() {
	gc.Cursor(0)

	e.stdscr.Clear()

	if e.x-e.printLineStartIndex > e.maxX-1 {
		e.printLineStartIndex = e.x - e.maxX + 1
	} else if e.x < e.printLineStartIndex {
		e.printLineStartIndex = e.x
	}

	if e.y-e.printLinesIndex > e.maxY-1 {
		e.printLinesIndex = e.y - e.maxY + 1
	} else if e.y < e.printLinesIndex {
		e.printLinesIndex = e.y
	}

	for i, line := range e.text[e.printLinesIndex:] {
		if i >= e.maxY {
			break
		}

		if len(line) <= e.printLineStartIndex {
			e.stdscr.Println()
			continue
		}
		line = line[e.printLineStartIndex:]
		if len(line) > e.maxX-1 {
			line = line[:e.maxX-1]
		}

		e.stdscr.Println(line)
	}

	e.stdscr.Move(e.y-e.printLinesIndex, e.x-e.printLineStartIndex)
	e.stdscr.Refresh()

	gc.Cursor(1)
}

func (e *Editor) Init() {
	var err error
	e.stdscr, err = gc.Init()

	if err != nil {
		log.Fatal("init", err)
	}

	gc.Echo(false)
	e.stdscr.Keypad(true)

	e.maxY, e.maxX = e.stdscr.MaxYX()

	e.text = make([]string, 1)
}
func (e *Editor) End() {
	gc.End()
}
func (e *Editor) Load(filePath string) error {
	lines, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	e.text = make([]string, 1)
	lineNr := 0
	for _, r := range lines {
		if r == 0x0D {
			continue
		}

		chr := string(r)

		if chr == "\n" {
			e.text = append(e.text, "")
			lineNr++
			continue
		}

		e.text[lineNr] += chr

	}
	e.y, e.x = e.stdscr.CursorYX()
	e.draw()

	e.currLengthIndex = e.x

	return nil
}
func (e *Editor) Run() error {
	for {
		key := e.stdscr.GetChar()

		updateLengthIndex := true

		switch key {
		case gc.KEY_ESC:
			return nil
		case gc.KEY_DOWN:
			if e.y >= len(e.text)-1 {
				continue
			}

			if e.currLengthIndex > len(e.text[e.y+1]) {
				e.x = len(e.text[e.y+1])
			} else {
				e.x = e.currLengthIndex
			}
			e.y++
			updateLengthIndex = false
		case gc.KEY_UP:
			if e.y <= 0 {
				continue
			}

			if e.currLengthIndex > len(e.text[e.y-1]) {
				e.x = len(e.text[e.y-1])
			} else {
				e.x = e.currLengthIndex
			}
			e.y--
			updateLengthIndex = false
		case gc.KEY_LEFT:
			if e.x <= 0 {
				if e.y > 0 {
					e.y--
					e.x = len(e.text[e.y])
				}
			} else {
				e.x--
			}
		case gc.KEY_RIGHT:
			if e.x >= len(e.text[e.y]) {
				if e.y < len(e.text)-1 {
					e.y++
					e.x = 0
				}
			} else {
				e.x++
			}
		case gc.KEY_ENTER, gc.KEY_RETURN:
			newLine := e.text[e.y][:e.x]
			e.text[e.y] = e.text[e.y][e.x:]

			before := make([]string, len(e.text[:e.y]))
			copy(before, e.text[:e.y])

			before = append(before, newLine)

			rest := make([]string, len(e.text[e.y:]))
			copy(rest, e.text[e.y:])

			e.text = append(before, rest...)

			e.y++
			e.x = 0
		case gc.KEY_TAB:
			e.text[e.y] = e.text[e.y][:e.x] + "    " + e.text[e.y][e.x:]
			e.x += 4
		case gc.KEY_END:
			e.x = len(e.text[e.y])
		case gc.KEY_HOME:
			e.x = 0
		case gc.KEY_BACKSPACE:
			if e.x == 0 {
				if e.y == 0 {
					continue
				}

				line := e.text[e.y]

				e.x = len(e.text[e.y-1])

				e.text[e.y-1] += line
				e.text = append(e.text[:e.y], e.text[e.y+1:]...)

				e.y--

			} else {
				e.x--
				e.text[e.y] = e.text[e.y][:e.x] + e.text[e.y][e.x+1:]
			}
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			e.text[e.y] = e.text[e.y][:e.x] + chr + e.text[e.y][e.x:]
			e.x++
		}

		if updateLengthIndex {
			e.currLengthIndex = e.x
		}

		e.draw()
	}
}
func (e *Editor) Save(filepath string) error {
	data := []byte(strings.Join(e.text, "\n"))
	err := os.WriteFile(filepath, data, 0666)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("missing argument {file}")
		os.Exit(1)
	}

	path := os.Args[1]

	e := &Editor{}
	e.Init()
	defer e.End()

	f, _ := os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(f)
	defer f.Close()

	if path != "" {
		_ = e.Load(path)
	}

	err := e.Run()
	if err != nil {
		panic(err)
	}
	if path != "" {
		err := e.Save(path)
		if err != nil {
			panic(err)
		}
	}
}
