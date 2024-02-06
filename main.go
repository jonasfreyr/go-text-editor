package main

import (
	"flag"
	gc "github.com/rthornton128/goncurses"
	"log"
	"os"
	"strings"
)

type Editor struct {
	text   []string
	stdscr *gc.Window

	maxX, maxY int
}

func (e *Editor) draw(y, x int) {
	e.stdscr.Clear()

	for i, line := range e.text {
		if i >= e.maxY-2 {
			break
		}

		e.stdscr.Println(line)
	}

	e.stdscr.Move(y, x)

	e.stdscr.Refresh()
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

	e.text = make([]string, e.maxY)
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
	y, x := e.stdscr.CursorYX()
	e.draw(y, x)

	return nil
}
func (e *Editor) Run() error {
	for {
		key := e.stdscr.GetChar()
		y, x := e.stdscr.CursorYX()

		switch key {
		case gc.KEY_ESC:
			return nil
		case gc.KEY_DOWN:
			if y >= len(e.text)-1 {
				continue
			}

			if x > len(e.text[y+1])-1 {
				x = len(e.text[y+1]) - 1
			}
			y++
		case gc.KEY_UP:
			if y <= 0 {
				continue
			}

			if x > len(e.text[y-1])-1 {
				x = len(e.text[y-1]) - 1
			}
			y--
		case gc.KEY_LEFT:
			if x <= 0 {
				if y > 0 {
					y--
					x = len(e.text[y])
				}
			}
			x--
		case gc.KEY_RIGHT:
			if x >= len(e.text[y]) {
				continue
			}
			x++
		case gc.KEY_ENTER, gc.KEY_RETURN:
			newLine := e.text[y][:x]
			e.text[y] = e.text[y][x:]

			before := make([]string, len(e.text[:y]))
			copy(before, e.text[:y])

			before = append(before, newLine)

			rest := make([]string, len(e.text[y:]))
			copy(rest, e.text[y:])

			e.text = append(before, rest...)

			y++
			x = 0

		case gc.KEY_TAB:
		case gc.KEY_BACKSPACE:
			if x == 0 {
				if y == 0 {
					continue
				}

				line := e.text[y]

				x = len(e.text[y-1])

				e.text[y-1] += line
				e.text = append(e.text[:y], e.text[y+1:]...)

				y--

			} else {
				x--
				e.text[y] = e.text[y][:x] + e.text[y][x+1:]
			}

		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			e.text[y] = e.text[y][:x] + chr + e.text[y][x:]
			x++
		}
		e.draw(y, x)
	}
}
func (e *Editor) Save(filepath string) error {
	data := []byte(strings.Join(e.text, ""))
	err := os.WriteFile(filepath, data, 0666)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	path := flag.String("filepath", "", "Path to the file you want to load")

	flag.Parse()

	e := &Editor{}
	e.Init()
	defer e.End()

	f, _ := os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(f)
	defer f.Close()

	if *path != "" {
		err := e.Load(*path)
		if err != nil {
			panic(err)
		}
	}

	err := e.Run()
	if err != nil {
		panic(err)
	}
	if *path != "" {
		// err := e.Save(*path)
		if err != nil {
			panic(err)
		}
	}
}
