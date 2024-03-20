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

	inlinePosition int

	printLineStartIndex int
	printLinesIndex     int

	lines    []string
	lexer    *Lexer
	colorMap map[string]int

	selectedXStart, selectedYStart int
	selectedXEnd, selectedYEnd     int
	selected                       string

	path string

	debug    bool
	debugscr *gc.Window

	miniWindow *MiniWindow
}

const TabWidth = 4

var colorIndex = 1

func (e *Editor) debugLog(args ...any) {
	if !e.debug {
		return
	}

	y, _ := e.debugscr.CursorYX()
	if y >= e.maxY {
		e.debugscr.Scroll(y - (e.maxY))
	}

	logString := ""
	for i, arg := range args {
		if i > 0 {
			e.debugscr.Print(" ")
			logString += " "
		}
		e.debugscr.Print(arg)
		logString += fmt.Sprint(arg)
	}
	e.debugscr.Println()
	log.Println(logString)
	e.debugscr.Refresh()
}
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
func (e *Editor) Init(debug bool) {
	var err error
	e.stdscr, err = gc.Init()

	if err != nil {
		log.Fatal("init", err)
	}

	e.maxY, e.maxX = e.stdscr.MaxYX()
	e.debug = debug
	if debug {
		e.stdscr, err = gc.NewWindow(e.maxY, e.maxX*3/5, 0, 4)
		if err != nil {
			log.Fatal(err)
		}

		e.debugscr, err = gc.NewWindow(e.maxY, e.maxX*3/5-4, 0, e.maxX*3/5+4)
		e.debugscr.ScrollOk(true)
		if err != nil {
			log.Fatal(err)
		}

		e.maxY, e.maxX = e.stdscr.MaxYX()

	} else {
		e.stdscr, err = gc.NewWindow(e.maxY, e.maxX, 0, 4)
		if err != nil {
			log.Fatal(err)
		}
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
	gc.Raw(true)       // Hell yeah
	gc.SetEscDelay(10) // Watch out for this

	err = e.stdscr.Keypad(true)
	if err != nil {
		log.Fatal(err)
	}

	gc.SetTabSize(TabWidth)

	//go func() {
	//	count := 0
	//	for {
	//		time.Sleep(time.Millisecond * 200)
	//		e.debugLog(fmt.Sprintf("test-%d", count))
	//		count++
	//	}
	//}()

	mw, err := gc.NewWindow(1, e.maxX, e.maxY-1, 4)
	if err != nil {
		log.Fatal(err)
	}

	err = mw.Keypad(true)
	if err != nil {
		log.Fatal(err)
	}

	e.miniWindow = &MiniWindow{
		width:  e.maxX,
		stdscr: mw,
	}

	e.lines = make([]string, 1)
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
	for i := 1; i <= e.maxY; i++ {
		e.lineNrscr.MovePrint(i-1, 0, fmt.Sprintf("%s", strconv.Itoa(start+i)))
	}
	e.disableColor(e.lineNrscr, e.lexer.config.LineNr.Color)
	e.lineNrscr.Refresh()
}
func (e *Editor) accountForTabs(x, y int) int {
	newX := 0
	for _, token := range e.lines[y][:x] {
		if string(token) == "\t" {
			newX += TabWidth - (newX % TabWidth)
		} else {
			newX++
		}
	}
	return newX
}
func (e *Editor) draw() {
	tokens := e.lexer.Tokenize(strings.Join(e.lines, "\n"))

	accountedForTabs := e.accountForTabs(e.x, e.y)

	// TODO: Don't know why it is 8 instead of 4
	if accountedForTabs-e.printLineStartIndex > e.maxX-TabWidth*2 {
		e.printLineStartIndex = accountedForTabs - e.maxX + TabWidth*2
	} else if accountedForTabs-TabWidth*2 < e.printLineStartIndex {
		e.printLineStartIndex = utils.Max(accountedForTabs-TabWidth*2, 0)
	}

	selectedXStart := e.accountForTabs(e.selectedXStart, e.selectedYStart)
	selectedXEnd := e.accountForTabs(e.selectedXEnd, e.selectedYEnd)
	selectedYStart := utils.Min(e.selectedYStart, e.selectedYEnd)
	selectedYEnd := utils.Max(e.selectedYStart, e.selectedYEnd)
	if selectedYStart == e.selectedYEnd { // Did the ends swap
		selectedXStart, selectedXEnd = selectedXEnd, selectedXStart
	}
	if selectedYStart == selectedYEnd { // Are the ends the same
		tempStart, tempEnd := selectedXStart, selectedXEnd
		selectedXStart = utils.Min(tempStart, tempEnd)
		selectedXEnd = utils.Max(tempStart, tempEnd)
	}

	// log.Println(selectedXStart, selectedYStart, selectedXEnd, selectedXEnd)

	err := gc.Cursor(0)
	if err != nil {
		log.Println(err)
	}

	e.drawLineNumbers()
	e.stdscr.Erase()
	e.selected = ""
	lastY := -1
	for i, line := range tokens[e.printLinesIndex:] {
		if i >= e.maxY {
			break
		}

		if len(line) == 0 || line[len(line)-1].location.col+line[len(line)-1].Length() <= e.printLineStartIndex {
			e.stdscr.Println()
			continue
		}

		for _, t := range line {
			x := t.location.col - e.printLineStartIndex
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
			e.stdscr.Move(i, x)
			for index, chr := range token {
				highlighted := false
				if e.isSelected(selectedXStart-e.printLineStartIndex, selectedXEnd-e.printLineStartIndex, selectedYStart, selectedYEnd, t.location.line, x+index) {
					highlighted = true
					e.disableColor(e.stdscr, t.color)
					e.stdscr.AttrOn(gc.A_REVERSE)
					if lastY != -1 && t.location.line != lastY {
						e.selected += "\n"
					}
					e.selected += string(chr)
					lastY = t.location.line
				}
				e.stdscr.AddChar(gc.Char(chr))

				if highlighted {
					e.enableColor(e.stdscr, t.color)
					e.stdscr.AttrOff(gc.A_REVERSE)
				}
			}
			// e.stdscr.Print(token)
			e.disableColor(e.stdscr, t.color)
		}
		e.stdscr.Println()

	}

	e.stdscr.Move(e.y-e.printLinesIndex, accountedForTabs-e.printLineStartIndex)

	e.stdscr.Refresh()

	if e.printLinesIndex <= e.y && e.y < e.printLinesIndex+e.maxY {
		err = gc.Cursor(1)
		if err != nil {
			log.Println(err)
		}
	}
}
func (e *Editor) End() {
	gc.End()
}
func (e *Editor) removeSelection() {
	selectedXStart := e.selectedXStart
	selectedXEnd := e.selectedXEnd
	selectedYStart := utils.Min(e.selectedYStart, e.selectedYEnd)
	selectedYEnd := utils.Max(e.selectedYStart, e.selectedYEnd)
	if selectedYStart == e.selectedYEnd { // Did the ends swap
		selectedXStart = e.selectedXEnd
		selectedXEnd = e.selectedXStart
	}
	if selectedYStart == selectedYEnd { // Are the ends the same
		selectedXStart = utils.Min(e.selectedXStart, e.selectedXEnd)
		selectedXEnd = utils.Max(e.selectedXStart, e.selectedXEnd)
		e.lines[e.y] = e.lines[e.y][:selectedXStart] + e.lines[e.y][selectedXEnd:]
		e.x = selectedXStart
		return
	}

	log.Println(selectedYStart, selectedXStart, selectedYEnd, selectedXEnd)
	log.Println(e.lines[selectedYEnd])

	// Please for the love of god fix this
	e.moveY(selectedYStart - e.y) // e.y = selectedYStart
	e.moveX(selectedXStart - e.x) // e.x = selectedXStart
	e.lines[selectedYStart] = e.lines[selectedYStart][:selectedXStart] + e.lines[selectedYEnd][selectedXEnd:]
	e.deleteLines(selectedYStart+1, selectedYEnd-selectedYStart)
}
func (e *Editor) remove(num int) {
	if e.x == 0 {
		if e.y == 0 {
			return
		}

		line := e.lines[e.y]

		e.x = len(e.lines[e.y-1])

		e.lines[e.y-1] += line

		e.lines = append(e.lines[:e.y], e.lines[e.y+num:]...)

		e.y--
		return
	}

	e.x -= num
	e.lines[e.y] = e.lines[e.y][:e.x] + e.lines[e.y][e.x+num:]
}
func (e *Editor) deleteLines(y, num int) {
	if len(e.lines) <= 0 {
		return
	} else if len(e.lines) == 1 {
		e.lines[y] = ""
	} else {
		e.lines = append(e.lines[:y], e.lines[utils.Min(y+num, len(e.lines)):]...)
	}
	// e.y = y
	e.clampX()
}
func (e *Editor) clampX() {
	line := e.lines[e.y]

	if e.inlinePosition > len(line) {
		e.x = len(line)
	} else {
		e.x = e.inlinePosition
	}
}
func (e *Editor) Load(filePath string) error {
	e.path = filePath

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

		text[lineNr] += chr

	}
	e.lines = text

	e.y, e.x = e.stdscr.CursorYX()

	before := time.Now()
	e.draw()
	dt := time.Since(before)
	log.Println(dt)

	e.inlinePosition = e.x

	return nil
}
func (e *Editor) moveY(delta int) {
	e.y = utils.Min(utils.Max(e.y+delta, 0), len(e.lines)-1)

	e.clampX()

	if e.y-e.printLinesIndex > e.maxY-TabWidth {
		e.printLinesIndex = e.y - e.maxY + TabWidth
	} else if e.y-TabWidth < e.printLinesIndex {
		e.printLinesIndex = utils.Max(e.y-TabWidth, 0)
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
			e.x += delta
		}
	} else {
		if e.x <= 0 {
			if e.y > 0 {
				e.moveY(-1)
				e.x = len(e.lines[e.y])
			}
		} else {
			e.x += delta
		}
	}
}
func (e *Editor) moveXto(x int) {
	e.moveY(x - e.x)
}
func (e *Editor) moveYto(y int) {
	e.moveY(y - e.y)
}
func (e *Editor) getTokenIndexByX(tokens []Token, x int) int {
	index := -1
	for i, token := range tokens {
		e.debugLog("X:", x, "token:", token.location.col, "tokenSize:", token.location.col+token.Length())
		if token.location.col <= x && token.location.col+token.Length() >= x {
			index = i
			e.debugLog("found")
			break
		}
	}
	e.debugLog("returning", index)
	e.debugLog("--------------------")
	return index
}
func filterSpacesAndTabs(tokens []Token) []Token {
	newTokens := make([]Token, 0)
	for _, token := range tokens {
		if token.lexeme != " " && token.lexeme != "\t" {
			newTokens = append(newTokens, token)
		}
	}
	return newTokens
}
func unAccountForTabs(tokens []Token) []Token {
	newTokens := make([]Token, 0)
	x := 0
	for _, token := range tokens {
		tok := Token{
			location: Location{
				line: token.location.line,
				col:  x,
			},
			lexeme: token.lexeme,
		}
		x += len(token.lexeme)
		newTokens = append(newTokens, tok)
	}

	return newTokens
}
func (e *Editor) ctrlMoveLeft() {
	if e.x == 0 {
		return
	}

	str := e.lines[e.y]
	tonkens := e.lexer.Tokenize(str)[0]
	tonkens = unAccountForTabs(tonkens)
	tonkens = filterSpacesAndTabs(tonkens)

	i := e.getTokenIndexByX(tonkens, e.x)

	var tonken Token
	if i == -1 {
		tonken = tonkens[len(tonkens)-1]

	} else {
		tonken = tonkens[i]
	}

	if tonken.location.col == e.x {
		if i == 0 {
			e.moveX(-e.x)
		} else {
			prevTonken := tonkens[i-1]
			e.moveX(prevTonken.location.col - e.x)
		}

	} else {
		e.moveX(tonken.location.col - e.x)
	}

	e.x = utils.Max(e.x, 0)
}
func (e *Editor) ctrlMoveRight() {
	str := e.lines[e.y]

	if len(str) == 0 {
		return
	}

	tonkens := e.lexer.Tokenize(str)[0]
	tonkens = unAccountForTabs(tonkens)
	tonkens = filterSpacesAndTabs(tonkens)

	i := e.getTokenIndexByX(tonkens, e.x)

	var tonken Token
	if i == -1 {
		tonken = tonkens[0]

	} else {
		tonken = tonkens[i]
	}

	if tonken.location.col+tonken.Length() == e.x && i != len(tonkens)-1 {
		nextTonken := tonkens[i+1]
		e.debugLog("next: ", nextTonken.location.col)
		e.moveX(nextTonken.location.col + nextTonken.Length() - e.x)
	} else {
		e.moveX(tonken.location.col + tonken.Length() - e.x)
	}
}
func (e *Editor) find(text string) {
	for lineNr, line := range e.lines {
		if index := strings.Index(line, text); index != -1 {
			e.debugLog("y, x", index, lineNr)
			e.moveYto(lineNr)
			e.moveXto(index)
			return
		}
	}
}
func (e *Editor) Run() error {
	for {
		err := e.run()

		if err != nil {
			e.debugLog(err)

		} else {
			return nil
		}
	}
}
func (e *Editor) run() error {
	for {
		key := e.stdscr.GetChar()

		e.debugLog(key, gc.KeyString(key))

		updateLengthIndex := true
		resetSelected := true
		currentLine := e.lines[e.y]

		switch key {
		case gc.KEY_ESC:
			return nil
		case 562, 566: // CTRL + Shift + Right
			e.ctrlMoveRight()
			resetSelected = false
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y
		case 561, 565: // CTRL + Right
			e.ctrlMoveRight()
		case 526, 530: // CTRL + Down
			e.printLinesIndex = utils.Min(e.printLinesIndex+1, len(e.lines))
		case 547, 551: // CTRL + Shift + Left
			e.ctrlMoveLeft()
			resetSelected = false
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y

		case 546, 550: // CTRL + Left
			e.ctrlMoveLeft()
		case 567, 571: // CTRL + Up
			e.printLinesIndex = utils.Max(e.printLinesIndex-1, 0)
		case 6: // CTRL + F
			str := e.miniWindow.run(true)
			e.find(str)
			e.debugLog("x is:", e.x)
			resetSelected = false
			e.selectedXStart = e.x
			e.selectedYStart = e.y
			e.selectedXEnd = e.x + len(str)
			e.selectedYEnd = e.y

		case 4: // CTRL + D
			e.deleteLines(e.y, 1)
			e.y = utils.Min(utils.Max(len(e.lines)-1, 0), utils.Max(e.y-1, 0))
			e.clampX()
		case 3: // CTRL + C
			text := e.selected
			if e.selected == "" {
				text = "\n" + currentLine
			}
			err := clipboard.WriteAll(text)
			if err != nil {
				panic(err)
			}
		case 1: // CTRL + A
			e.selectedYStart = 0
			e.selectedXStart = 0
			e.selectedXEnd = len(e.lines[len(e.lines)-1])
			e.selectedYEnd = len(e.lines) - 1

			e.moveY(e.selectedYEnd - e.y)
			e.moveX(e.selectedXEnd - e.x)
			//e.x = e.selectedXEnd
			//e.y = e.selectedYEnd
			resetSelected = false
		case 24: // CTRL + X
			var text string
			if e.selected == "" {
				text = "\n" + currentLine
				e.deleteLines(e.y, 1)
			} else {
				text = e.selected
				e.removeSelection()
			}

			err := clipboard.WriteAll(text)

			if err != nil {
				panic(err)
			}
		case 19: // CTRL + S
			err := e.Save(e.path)
			if err != nil {
				log.Println(err)
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
		case 536: // CTRL+Home
			e.moveY(-e.y)
			updateLengthIndex = false
		case 531: // CTRL+END
			e.moveY(len(e.lines) - e.y)
			updateLengthIndex = false
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
			e.lines[e.y] = e.lines[e.y][:e.x] + "\t" + e.lines[e.y][e.x:]
			e.moveX(1)
		case gc.KEY_SEND:
			e.x = len(e.lines[e.y])
			resetSelected = false
			e.selectedXEnd = e.x
		case gc.KEY_END:
			e.x = len(e.lines[e.y])
		case gc.KEY_SHOME:
			e.x = 0
			resetSelected = false
			e.selectedXEnd = e.x
		case gc.KEY_HOME:
			e.x = 0
		case gc.KEY_BACKSPACE:
			if e.selected != "" {
				e.removeSelection()
				break
			}

			e.remove(1)
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			e.lines[e.y] = e.lines[e.y][:e.x] + chr + e.lines[e.y][e.x:]
			e.x++
		}

		if updateLengthIndex {
			e.inlinePosition = e.x
		}
		if resetSelected {
			e.selectedXStart = e.x
			e.selectedYStart = e.y
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y
		}

		e.y = utils.Min(utils.Max(len(e.lines)-1, 0), e.y)

		e.draw()
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
	debug := false
	if len(os.Args) >= 3 && os.Args[2] == "--debug" {
		debug = true
	}

	e := &Editor{}
	e.Init(debug)
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
}
