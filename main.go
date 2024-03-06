package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
)

type Editor struct {
	stdscr *DoubleBufferWindow

	maxX, maxY int

	x, y int
	visX int

	currLengthIndex int

	printLineStartIndex int
	printLinesIndex     int

	lines    []string
	lexer    *Lexer
	colorMap map[string]int
}

func tokenLineLength(tokens []Token) int {
	l := 0
	for _, token := range tokens {
		l += token.Length()
	}
	return l
}

func (e *Editor) disableColor(color [3]int) {
	key := utils.ColorToString(color)
	colorIndex := e.colorMap[key]
	e.stdscr.ColorOff(int16(colorIndex))
}

func (e *Editor) enableColor(color [3]int) {
	key := utils.ColorToString(color)
	colorIndex := e.colorMap[key]
	e.stdscr.ColorOn(int16(colorIndex))
}

func (e *Editor) accountForTabs(y, x int) int {
	extra := 0
	for col, l := range e.lines[y][:x] {
		if string(l) == "\t" {
			extra += 4 - (col)%4
		}
	}

	return utils.Max(extra-1, 0)
}

func (e *Editor) draw(swap bool) {
	tokens := e.lexer.Tokenize(strings.Join(e.lines, "\n"))

	if e.x-e.printLineStartIndex > e.maxX-4 {
		e.printLineStartIndex = e.x - e.maxX + 4
	} else if e.x-4 < e.printLineStartIndex {
		e.printLineStartIndex = utils.Max(e.x-4, 0)
	}

	if e.y-e.printLinesIndex > e.maxY-4 {
		e.printLinesIndex = e.y - e.maxY + 4
	} else if e.y-4 < e.printLinesIndex {
		e.printLinesIndex = utils.Max(e.y-4, 0)
	}

	err := gc.Cursor(0)
	if err != nil {
		log.Println(err)
	}

	err = e.stdscr.Clear()
	if err != nil {
		log.Println(err)
	}
	for i, line := range tokens[e.printLinesIndex:] {
		if i >= e.maxY {
			break
		}

		if tokenLineLength(line) <= e.printLineStartIndex {
			e.stdscr.Println()
			continue
		}

		for _, t := range line {
			x := t.location.col - 1 - e.printLineStartIndex
			token := t.Token()

			// Either skip or cut tokens that are not on screen to the left
			if x < 0 {
				if x+len(token) < 0 {
					continue
				}

				token = token[-x:]
				x = 0
			}

			// Either skip or cut tokens that are not on screen to the right
			maxX := e.maxX - 1
			if x+len(token) > maxX {
				if x > maxX {
					break
				}

				token = token[:maxX-x]
			}

			e.enableColor(t.color)
			e.stdscr.Print(token)
			e.disableColor(t.color)
		}
		e.stdscr.Println()

	}

	e.stdscr.Move(e.y-e.printLinesIndex, e.x-e.printLineStartIndex+e.accountForTabs(e.y, e.x))
	if swap {
		e.stdscr.Refresh()
	} else {
		e.stdscr.NRefresh()
	}
	err = gc.Cursor(1)
	if err != nil {
		log.Println(err)
	}
}
func (e *Editor) setColor(index int, color [3]int) error {
	// log.Println("Setting color", index, color)
	err := gc.InitColor(int16(index), int16(utils.MapTo1000(color[0])), int16(utils.MapTo1000(color[1])), int16(utils.MapTo1000(color[2])))
	if err != nil {
		return err
	}

	// fmt.Println("Setting pair")
	err = gc.InitPair(int16(index), int16(index), -1)
	if err != nil {
		return err
	}

	key := utils.ColorToString(color)
	e.colorMap[key] = index

	return nil
}
func (e *Editor) setColors() error {
	e.colorMap = make(map[string]int)
	for i, colorArray := range [][3]int{e.lexer.config.Literals.Color, e.lexer.config.BuiltIns.Color, e.lexer.config.Types.Color,
		e.lexer.config.Keywords.Color, e.lexer.config.Comment.Color, e.lexer.config.Digits.Color, e.lexer.config.Strings.Color, e.lexer.config.Default.Color} {
		err := e.setColor(i+1, colorArray)
		if err != nil {
			return err
		}
	}

	return nil
}
func (e *Editor) initColor() error {
	if !gc.HasColors() {
		return nil
	}

	err := gc.StartColor()
	if err != nil {
		return err
	}

	err = gc.UseDefaultColors()
	if err != nil {
		return err
	}

	err = e.setColors()
	if err != nil {
		return err
	}
	return nil
}
func (e *Editor) Init() {
	var err error
	e.stdscr, err = NewDoubleBufferWindow()

	if err != nil {
		log.Fatal("init", err)
	}

	e.lexer = NewLexer()

	err = e.initColor()
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	gc.Echo(false)
	err = e.stdscr.Keypad(true)
	if err != nil {
		log.Fatal(err)
	}

	gc.SetTabSize(4)

	e.maxY, e.maxX = e.stdscr.MaxYX()

	e.lines = make([]string, 1)
}
func (e *Editor) End() {
	gc.End()
}

// This needs testing when it comes to multi line deletes
func (e *Editor) deleteLines(y, num int) {
	if len(e.lines) <= 0 {
		return
	} else if len(e.lines) == 1 {
		e.lines[y] = ""
	} else {
		e.lines = append(e.lines[:y], e.lines[y+num:]...)
	}
}
func (e *Editor) clampXToLineOrLengthIndex() {
	line := e.lines[e.y]
	if e.currLengthIndex > len(line) {
		e.x = len(line)
	} else {
		e.x = e.currLengthIndex
	}
}
func (e *Editor) Load(filePath string) error {
	lines, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	text := make([]string, 1)
	lineNr := 0
	for _, r := range lines {
		if r == 0x0D {
			continue
		}

		chr := string(r)

		if chr == "\n" {
			text = append(text, "")
			lineNr++
			continue
		}

		//if chr == "\t" {
		//	chr = "    "
		//}

		text[lineNr] += chr

	}
	e.lines = text

	e.y, e.x = e.stdscr.CursorYX()

	before := time.Now()
	e.draw(false)
	dt := time.Since(before)
	log.Println(dt)

	e.currLengthIndex = e.x

	return nil
}

func (e *Editor) Run() error {
	for {
		key := e.stdscr.GetChar()

		updateLengthIndex := true
		currentLine := e.lines[e.y]

		switch key {
		case gc.KEY_ESC:
			return nil
		case 561: // CTRL + Right
			str := e.lines[e.y][e.x:]
			i := strings.Index(str, " ")
			if i == -1 {
				e.x = len(e.lines[e.y])
			} else if i == 0 {
				e.x++
			} else {
				e.x = e.x + i
			}
		case 526: // CTRL + Down
		case 546: // CTRL + Left
			str := e.lines[e.y][:e.x]
			i := strings.LastIndex(str, " ")
			if i == -1 {
				e.x = 0
			} else if i == len(str)-1 {
				e.x--
			} else {
				e.x = i + 1
			}
		case 567: // CTRL + Up
		case 4: // CTRL + D
			e.deleteLines(e.y, 1)
			e.y--
			e.y = utils.Min(utils.Max(len(e.lines)-1, 0), utils.Max(e.y, 0))
			e.clampXToLineOrLengthIndex()
		case 24: // CTRL + X
			text := currentLine + "\n"
			err := clipboard.WriteAll(text)
			e.deleteLines(e.y, 1)
			if err != nil {
				panic(err)
			}
		case gc.KEY_DOWN:
			if e.y >= len(e.lines)-1 {
				continue
			}

			e.y++
			e.clampXToLineOrLengthIndex()
			updateLengthIndex = false
		case gc.KEY_UP:
			if e.y <= 0 {
				continue
			}

			e.y--
			e.clampXToLineOrLengthIndex()
			updateLengthIndex = false
		case gc.KEY_LEFT:
			if e.x <= 0 {
				if e.y > 0 {
					e.y--
					e.x = len(e.lines[e.y])
				}
			} else {
				e.x--
			}
		case gc.KEY_RIGHT:
			if e.x >= len(currentLine) {
				if e.y < len(e.lines)-1 {
					e.y++
					e.x = 0
				}
			} else {
				e.x++
			}
		case gc.KEY_ENTER, gc.KEY_RETURN:
			newLine := e.lines[e.y][:e.x]
			e.lines[e.y] = e.lines[e.y][e.x:]

			before := make([]string, len(e.lines[:e.y]))
			copy(before, e.lines[:e.y])

			before = append(before, newLine)

			rest := make([]string, len(e.lines[e.y:]))
			copy(rest, e.lines[e.y:])

			e.lines = append(before, rest...)

			e.y++
			e.x = 0
		case gc.KEY_TAB:
			e.lines[e.y] = e.lines[e.y][:e.x] + "\t" + e.lines[e.y][e.x:]
			e.x++
		case gc.KEY_END:
			e.x = len(e.lines[e.y])
		case gc.KEY_HOME:
			e.x = 0
		case gc.KEY_BACKSPACE:
			if e.x == 0 {
				if e.y == 0 {
					continue
				}

				line := e.lines[e.y]

				e.x = len(e.lines[e.y-1])

				e.lines[e.y-1] += line
				e.lines = append(e.lines[:e.y], e.lines[e.y+1:]...)

				e.y--

			} else {
				e.x--
				e.lines[e.y] = e.lines[e.y][:e.x] + e.lines[e.y][e.x+1:]
			}
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			e.lines[e.y] = e.lines[e.y][:e.x] + chr + e.lines[e.y][e.x:]
			e.x++
		}

		if updateLengthIndex {
			e.currLengthIndex = e.x
		}

		e.y = utils.Min(utils.Max(len(e.lines)-1, 0), e.y)

		e.draw(true)
	}
}
func (e *Editor) Save(filepath string) error {
	data := []byte(strings.Join(e.lines, "\n"))
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

	f, err := os.Create("logs.txt")
	if err != nil {
		panic(err)
	}

	log.SetOutput(f)
	defer f.Close()

	if path != "" {
		_ = e.Load(path)
	}

	err = e.Run()
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
