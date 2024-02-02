package main

import (
	"flag"
	"log"
	"os"
	"strings"

	gc "github.com/rthornton128/goncurses"
)

type Editor struct {
	text   []string
	stdscr *gc.Window
}

func (e *Editor) draw() {
	e.stdscr.Clear()

	y, x := e.stdscr.CursorYX()

	log.Println(x, y)

	e.stdscr.Move(0, 0)

	for _, line := range e.text {
		e.stdscr.Print(line)
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

	maxY, _ := e.stdscr.MaxYX()

	e.text = make([]string, maxY)
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
		e.text[lineNr] += chr

		if chr == "\n" {
			e.text = append(e.text, "")
			lineNr++
		}
	}

	e.draw()

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

			e.stdscr.Move(y+1, x)

		case gc.KEY_UP:
			if y <= 0 {
				continue
			}

			if x > len(e.text[y-1])-1 {
				x = len(e.text[y-1]) - 1
			}

			e.stdscr.Move(y-1, x)
		case gc.KEY_LEFT:
			if x <= 0 {
				continue
			}

			e.stdscr.Move(y, x-1)

		case gc.KEY_RIGHT:
			if x >= len(e.text[y])-1 {
				continue
			}

			e.stdscr.Move(y, x+1)

		case gc.KEY_ENTER, gc.KEY_RETURN:
			e.text[y] = e.text[y][:x] + "\n" + e.text[y][x:]

			if y+1 >= len(e.text) {
				e.text = append(e.text, "")
			}

			e.stdscr.Move(y+1, x)

		case gc.KEY_TAB:
		case gc.KEY_BACKSPACE:
			if x == 0 {
				if y == 0 {
					continue
				}
				y -= 1
				x = len(e.text[y]) - 1
			} else {
				x--
			}
			e.stdscr.Move(y, x)
			err := e.stdscr.DelChar()
			e.text[y] = e.text[y][:x] + e.text[y][x+1:]
			if err != nil {
				return err
			}
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			e.text[y] = e.text[y][:x] + chr + e.text[y][x:]
		}
		e.draw()
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
		err := e.Save(*path)
		if err != nil {
			panic(err)
		}
	}
}
