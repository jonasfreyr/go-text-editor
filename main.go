package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
)

type Editor struct {
	stdscr    *gc.Window
	lineNrscr *gc.Window

	maxX, maxY int

	x, y int
	visX int

	currLengthIndex int

	printLineStartIndex int
	printLinesIndex     int

	lines    []string
	lexer    *Lexer
	colorMap map[string]int

	selectedXStart, selectedYStart int
	selectedXEnd, selectedYEnd     int
}

var colorIndex = 1

func (e *Editor) setColor(color [3]int) error {
	// log.Println("Setting color", index, color)
	err := gc.InitColor(int16(colorIndex), int16(utils.MapTo1000(color[0])), int16(utils.MapTo1000(color[1])), int16(utils.MapTo1000(color[2])))
	if err != nil {
		return err
	}

	// fmt.Println("Setting pair")
	err = gc.InitPair(int16(colorIndex), int16(colorIndex), -1)
	if err != nil {
		return err
	}

	key := utils.ColorToString(color)
	e.colorMap[key] = colorIndex

	colorIndex++

	return nil
}
func (e *Editor) setColors() error {
	e.colorMap = make(map[string]int)
	for _, colorArray := range [][3]int{e.lexer.config.Literals.Color, e.lexer.config.BuiltIns.Color, e.lexer.config.Types.Color, e.lexer.config.LineNr.Color,
		e.lexer.config.Keywords.Color, e.lexer.config.Comment.Color, e.lexer.config.Digits.Color, e.lexer.config.Strings.Color, e.lexer.config.Default.Color} {
		err := e.setColor(colorArray)
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
	//e.stdscr, err = NewDoubleBufferWindow()
	e.stdscr, err = gc.Init()

	if err != nil {
		log.Fatal("init", err)
	}

	e.maxY, e.maxX = e.stdscr.MaxYX()

	e.stdscr, err = gc.NewWindow(e.maxY, e.maxX, 0, 4)
	if err != nil {
		log.Fatal(err)
	}

	e.lineNrscr, err = gc.NewWindow(e.maxY, 4, 0, 0)
	if err != nil {
		log.Fatal(err)
	}

	e.lexer = NewLexer()

	err = e.initColor()
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	gc.Echo(false)
	gc.Raw(true) // Hell yeah
	err = e.stdscr.Keypad(true)
	if err != nil {
		log.Fatal(err)
	}

	gc.SetTabSize(4)

	e.lines = make([]string, 1)
}
func tokenLineLength(tokens []Token) int {
	l := 0
	for _, token := range tokens {
		l += token.Length()
	}
	return l
}
func (e *Editor) disableColor(scr *gc.Window, color [3]int) {
	key := utils.ColorToString(color)
	colorIndex := e.colorMap[key]
	err := scr.ColorOff(int16(colorIndex))
	if err != nil {
		log.Println(err)
	}
}
func (e *Editor) enableColor(scr *gc.Window, color [3]int) {
	key := utils.ColorToString(color)
	colorIndex := e.colorMap[key]
	err := scr.ColorOn(int16(colorIndex))
	if err != nil {
		log.Println(err)
	}
}
func (e *Editor) isSelected(startX, endX, startY, endY, line, col int) bool {
	if startX == endX && startY == endY {
		return false
	}

	if startY < line && line < endY {
		return true
	}

	if startY != endY {
		if line == startY {
			return col >= startX
		} else if line == endY {
			return col < endX
		}
		return false
	}

	return (line >= startY && line <= endY) && (col >= startX && col < endX)
}
func (e *Editor) drawLineNumbers() {
	start := e.printLinesIndex
	e.lineNrscr.Erase()
	e.enableColor(e.lineNrscr, e.lexer.config.LineNr.Color)
	for i := 0; i < e.maxY; i++ {
		e.lineNrscr.MovePrint(i, 0, fmt.Sprintf("%s", strconv.Itoa(start+i)))
	}
	e.disableColor(e.lineNrscr, e.lexer.config.LineNr.Color)
	e.lineNrscr.Refresh()
}
func (e *Editor) draw(swap bool) {
	tokens := e.lexer.Tokenize(strings.Join(e.lines, "\n"))

	if e.x-e.printLineStartIndex > e.maxX-4 {
		e.printLineStartIndex = e.x - e.maxX + 4
	} else if e.x-4 < e.printLineStartIndex {
		e.printLineStartIndex = utils.Max(e.x-4, 0)
	}

	selectedXStart := e.selectedXStart
	selectedYStart := utils.Min(e.selectedYStart, e.selectedYEnd)
	selectedXEnd := e.selectedXEnd
	selectedYEnd := utils.Max(e.selectedYStart, e.selectedYEnd)
	if selectedYStart == e.selectedYEnd {
		selectedXStart = e.selectedXEnd
		selectedXEnd = e.selectedXStart

		if selectedYStart == selectedYEnd {
			selectedXStart = utils.Min(e.selectedXStart, e.selectedXEnd)
			selectedXEnd = utils.Max(e.selectedXStart, e.selectedXEnd)
		}
	}

	// log.Println(selectedXStart, selectedYStart, selectedXEnd, selectedXEnd)

	err := gc.Cursor(0)
	if err != nil {
		log.Println(err)
	}

	e.drawLineNumbers()
	e.stdscr.Erase()
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

			e.enableColor(e.stdscr, t.color)
			// e.stdscr.Move(i, x)
			for index, chr := range token {
				highlighted := false
				if e.isSelected(selectedXStart, selectedXEnd, selectedYStart, selectedYEnd, t.location.line, x+index) {
					highlighted = true
					e.stdscr.AttrOn(gc.A_REVERSE)
				}
				e.stdscr.AddChar(gc.Char(chr))

				if highlighted {
					e.stdscr.AttrOff(gc.A_REVERSE)
				}
			}
			// e.stdscr.Print(token)
			e.disableColor(e.stdscr, t.color)
		}
		e.stdscr.Println()

	}

	e.stdscr.Move(e.y-e.printLinesIndex, e.x-e.printLineStartIndex)

	e.stdscr.Refresh()

	if e.printLinesIndex <= e.y && e.y <= e.printLinesIndex+e.maxY {
		err = gc.Cursor(1)
		if err != nil {
			log.Println(err)
		}
	}
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

		if chr == "\t" {
			chr = "    "
		}

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
func (e *Editor) moveY(delta int) {
	e.y = utils.Min(utils.Max(e.y+delta, 0), len(e.lines)-1)

	e.clampXToLineOrLengthIndex()

	if e.y-e.printLinesIndex > e.maxY-4 {
		e.printLinesIndex = e.y - e.maxY + 4
	} else if e.y-4 < e.printLinesIndex {
		e.printLinesIndex = utils.Max(e.y-4, 0)
	}
}
func (e *Editor) moveX(delta int) {
	if delta > 0 {
		if e.x >= len(e.lines[e.y]) {
			if e.y < len(e.lines)-1 {
				e.moveY(1)
				e.x = 0
			}
		} else {
			e.x++
		}
	} else {
		if e.x <= 0 {
			if e.y > 0 {
				e.moveY(-1)
				e.x = len(e.lines[e.y])
			}
		} else {
			e.x--
		}
	}
}
func (e *Editor) Run() error {
	for {
		key := e.stdscr.GetChar()

		log.Println(key, gc.KeyString(key))

		updateLengthIndex := true
		resetSelected := true
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
		case 526, 530: // CTRL + Down
			e.printLinesIndex = utils.Min(e.printLinesIndex+1, len(e.lines))
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
		case 567, 571: // CTRL + Up
			e.printLinesIndex = utils.Max(e.printLinesIndex-1, 0)
		case 5: // CTRL + E

		case 4: // CTRL + D
			e.deleteLines(e.y, 1)
			e.y = utils.Min(utils.Max(len(e.lines)-1, 0), utils.Max(e.y-1, 0))
			e.clampXToLineOrLengthIndex()
		case 3: // CTRL + C

		case 24: // CTRL + X
			text := currentLine + "\n"
			err := clipboard.WriteAll(text)
			e.deleteLines(e.y, 1)
			if err != nil {
				panic(err)
			}
		case 337: // Shift+Up
			e.moveY(-1)
			updateLengthIndex = false
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case 402: // Shift+Right
			e.moveX(1)
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case 336: // Shift+Down
			e.moveY(1)
			updateLengthIndex = false
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x

			// log.Println(e.selectedYEnd, e.selectedYEnd)
		case 393: // Shift+Left
			e.moveX(-1)
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case gc.KEY_DOWN:
			e.moveY(1)
			updateLengthIndex = false
		case gc.KEY_UP:
			e.moveY(-1)
			updateLengthIndex = false
		case gc.KEY_LEFT:
			e.moveX(-1)
		case gc.KEY_RIGHT:
			e.moveX(1)
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
			e.lines[e.y] = e.lines[e.y][:e.x] + "    " + e.lines[e.y][e.x:]
			e.x += 4
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
		if resetSelected {
			e.selectedXStart = e.x
			e.selectedYStart = e.y
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y
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
