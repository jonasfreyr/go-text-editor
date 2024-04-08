package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	headerscr *gc.Window
	// unsavedscr *gc.Window

	maxX, maxY int

	x, y int
	visX int

	inlinePosition int

	printLineStartIndex int
	printLinesIndex     int

	lines []string
	lexer *Lexer

	selectedXStart, selectedYStart int
	selectedXEnd, selectedYEnd     int
	selected                       string

	path string

	debugscr *gc.Window

	miniWindow  *MiniWindow
	menuWindow  *MenuWindow
	popupWindow *PopUpWindow

	transactions *Transactions

	config *EditorConfig

	// TODO: Maybe collect all these into a struct
	openPathsToNames map[string]string // paths to name
	openedFiles      []string          // List of paths
	modified         map[string]bool   // paths to bool
	current          int
	tempFilePaths    map[string]string // paths to temp file paths
	tempFilePos      map[string]Location
}

var DEBUG_MODE = false

func (e *Editor) debugLog(args ...any) {
	if !DEBUG_MODE {
		return
	}

	// e.debugscr.Border(gc.ACS_VLINE, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS, gc.A_INVIS)

	err := gc.Cursor(0)
	if err != nil {
		log.Println(err)
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

	accountedForTabs := e.accountForTabs(e.x, e.y)
	e.stdscr.Move(e.y-e.printLinesIndex, accountedForTabs-e.printLineStartIndex)

	err = gc.Cursor(1)
	if err != nil {
		log.Println(err)
	}
}

func (e *Editor) Init() {
	var err error
	e.stdscr, err = gc.Init()

	if err != nil {
		log.Fatal("init", err)
	}

	e.config, err = ReadEditorConfig()
	if err != nil {
		e.debugLog(err)
	}

	// TODO: I hate this
	TabWidth = e.config.TabWidth

	e.maxY, e.maxX = e.stdscr.MaxYX()
	if DEBUG_MODE {
		e.stdscr, err = gc.NewWindow(e.maxY, e.maxX*3/5, 2, e.config.LineNumberWidth)
		if err != nil {
			e.End()
			log.Fatal(err)
		}

		e.debugscr, err = gc.NewWindow(e.maxY, e.maxX*3/5-e.config.LineNumberWidth, 0, e.maxX*3/5+e.config.LineNumberWidth)
		e.debugscr.ScrollOk(true)
		if err != nil {
			e.End()
			log.Fatal(err)
		}

		e.maxY, e.maxX = e.stdscr.MaxYX()

	} else {
		e.stdscr, err = gc.NewWindow(e.maxY, e.maxX-e.config.LineNumberWidth, 2, e.config.LineNumberWidth)
		e.maxY, e.maxX = e.stdscr.MaxYX()
		if err != nil {
			e.End()
			log.Fatal(err)
		}
	}

	e.lineNrscr, err = gc.NewWindow(e.maxY, e.config.LineNumberWidth, 2, 0)
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	e.headerscr, err = gc.NewWindow(2, e.maxX+e.config.LineNumberWidth, 0, 0)
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	//e.unsavedscr, err = gc.NewWindow(2, e.config.LineNumberWidth-1, 0, 0)
	//if err != nil {
	//	e.End()
	//	log.Fatal(err)
	//}

	e.lexer, err = NewLexer()
	if err != nil {
		e.End()
		log.Fatal("failed to load lexer: ", err)
	}

	err = InitColor()
	if err != nil {
		log.Println("failed to initialize color:", err)
	}

	gc.Echo(false)
	gc.Raw(true)       // Hell yeah
	gc.SetEscDelay(10) // Watch out for this

	err = e.stdscr.Keypad(true)
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	gc.SetTabSize(e.config.TabWidth)

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
		e.End()
		log.Fatal(err)
	}

	err = mw.Keypad(true)
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	e.miniWindow = &MiniWindow{
		width:  e.maxX,
		stdscr: mw,
		texts:  make(map[string]string),
	}

	e.lines = make([]string, 1)

	e.transactions = NewTransactions()

	// TODO: not hardcode these values
	width := e.maxX - 12
	height := 20
	e.menuWindow, err = NewMenuWindow(e.maxY/2-(height/2), utils.Max(e.maxX/2-(width/2), 4), height, width)
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	e.openPathsToNames = make(map[string]string)
	e.openedFiles = make([]string, 0)
	e.modified = make(map[string]bool)
	e.tempFilePaths = make(map[string]string)
	e.tempFilePos = make(map[string]Location)

	e.popupWindow, err = NewPopUpWindow(e.maxY/2, e.maxX/2, 3, 5)
	if err != nil {
		e.End()
		log.Fatal(err)
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
func (e *Editor) drawHeader() {
	e.headerscr.Erase()
	e.headerscr.HLine(1, 0, 0, e.maxX+e.config.LineNumberWidth-1)
	e.headerscr.MoveAddChar(1, e.config.LineNumberWidth-1, gc.ACS_TTEE)
	e.headerscr.Move(0, 0)

	for _, path := range e.openedFiles {
		//e.debugLog("drawing: ", path)
		name := e.openPathsToNames[path]
		if e.modified[path] {
			name = "*" + name
		}

		if path == e.path {
			e.headerscr.AttrOn(gc.A_REVERSE)
			e.headerscr.Print(name)
			e.headerscr.AttrOff(gc.A_REVERSE)
		} else {
			e.headerscr.Print(name)
		}
		_, x := e.headerscr.CursorYX()
		e.headerscr.VLine(0, x, 0, 1)
		// e.headerscr.MoveAddChar(1, x, gc.ACS_BTEE)
		e.headerscr.Move(0, x+1)
		//e.headerscr.Print(" ")
	}
	e.headerscr.VLine(0, e.maxX+e.config.LineNumberWidth-1, 0, 1)

	if DEBUG_MODE {
		e.headerscr.MoveAddChar(1, e.maxX+e.config.LineNumberWidth-1, gc.ACS_RTEE)
	} else {
		e.headerscr.MoveAddChar(1, e.maxX+e.config.LineNumberWidth-1, gc.ACS_LRCORNER)
	}
	e.headerscr.Refresh()
}
func (e *Editor) drawLineNumbers() {
	start := e.printLinesIndex
	e.lineNrscr.Erase()
	EnableColor(e.lineNrscr, e.config.LineNumberColor.Color)
	for i := 1; i <= e.maxY; i++ {
		e.lineNrscr.MovePrint(i-1, 0, fmt.Sprintf("%s", strconv.Itoa(start+i)))
	}
	DisableColor(e.lineNrscr, e.config.LineNumberColor.Color)
	e.lineNrscr.VLine(0, e.config.LineNumberWidth-1, 0, e.maxY)
	e.lineNrscr.Refresh()
}

func (e *Editor) accountForTabs(x, y int) int {
	newX := 0

	if x > len(e.lines[y]) {
		x = len(e.lines[y])
	}

	for _, token := range e.lines[y][:x] {
		if string(token) == "\t" {
			newX += e.config.TabWidth - (newX % e.config.TabWidth)
		} else {
			newX++
		}
	}
	return newX
}
func (e *Editor) draw() {
	// before := time.Now()

	accountedForTabs := e.accountForTabs(e.x, e.y)

	// TODO: Don't know why it is 8 instead of 4
	if accountedForTabs-e.printLineStartIndex > e.maxX-e.config.TabWidth*2 {
		e.printLineStartIndex = accountedForTabs - e.maxX + e.config.TabWidth*2
	} else if accountedForTabs-e.config.TabWidth*2 < e.printLineStartIndex {
		e.printLineStartIndex = utils.Max(accountedForTabs-e.config.TabWidth*2, 0)
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

	err := gc.Cursor(0)
	if err != nil {
		e.debugLog(err)
	}
	e.drawLineNumbers()
	e.drawHeader()
	// e.drawUnsaved()
	e.stdscr.Erase()
	e.selected = ""
	lastY := -1

	tokens := e.lexer.Tokenize(strings.Join(e.lines[:utils.Min(e.printLinesIndex+e.maxY, len(e.lines))], "\n"))

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

			EnableColor(e.stdscr, t.color)
			e.stdscr.Move(i, x)
			for index, chr := range token {
				highlighted := false
				if e.isSelected(selectedXStart-e.printLineStartIndex, selectedXEnd-e.printLineStartIndex, selectedYStart, selectedYEnd, t.location.line, x+index) {
					highlighted = true
					DisableColor(e.stdscr, t.color)
					e.stdscr.AttrOn(gc.A_REVERSE)
					if lastY != -1 && t.location.line != lastY {
						e.selected += "\n"
					}
					e.selected += string(chr)
					lastY = t.location.line
				}
				e.stdscr.AddChar(gc.Char(chr))

				if highlighted {
					EnableColor(e.stdscr, t.color)
					e.stdscr.AttrOff(gc.A_REVERSE)
				}
			}
			// e.stdscr.Print(token)
			DisableColor(e.stdscr, t.color)
		}
		e.stdscr.Println()

	}
	if DEBUG_MODE {
		e.stdscr.VLine(0, e.maxX-1, 0, e.maxY)
	}

	e.stdscr.Move(e.y-e.printLinesIndex, accountedForTabs-e.printLineStartIndex)

	if e.printLinesIndex <= e.y && e.y < e.printLinesIndex+e.maxY {
		err = gc.Cursor(1)
		if err != nil {
			e.debugLog(err)
		}
	}

	e.stdscr.Refresh()

	// dt := time.Since(before)
	// e.debugLog("draw time:", dt)
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
		e.remove(selectedYStart, selectedXEnd, selectedXEnd-selectedXStart)
		// e.lines[selectedYStart] = e.lines[selectedYStart][:selectedXStart] + e.lines[selectedYStart][selectedXEnd:]
		e.moveXto(selectedXStart)
		return
	}

	text := e.lines[selectedYEnd][selectedXEnd:]
	e.remove(selectedYStart, len(e.lines[selectedYStart]), len(e.lines[selectedYStart])-selectedXStart)
	e.insert(selectedYStart, selectedXStart, text)
	e.deleteLines(selectedYStart+1, selectedYEnd-selectedYStart)
	e.moveYto(selectedYStart) // e.y = selectedYStart
	e.moveXto(selectedXStart) // e.x = selectedXStart
}

// Removes num amount of characters starting from x on line y, if num is more than the characters then the line is removed
// If you desire to remove multiple lines use deleteLines
func (e *Editor) removeText(y, x, num int) string {
	text := e.lines[y][x-num : x]
	x -= num

	e.lines[y] = e.lines[y][:x] + e.lines[y][x+num:]
	return text
}
func (e *Editor) remove(y, x, num int) {
	e.modified[e.path] = true

	// TODO: will needs some fixing to work with nums larger than 1
	if x == 0 {
		if y == 0 {
			return
		}

		line := e.lines[y]

		e.insert(y-1, len(e.lines[y-1]), line)
		e.deleteLines(y, 1)
		return
	}

	text := e.removeText(y, x, num)

	if text != "" {
		ta := Action{
			location: Location{
				line: y,
				col:  x - num,
			},
			actionType: DELETE,
			text:       text,
		}
		e.transactions.addAction(ta)
	}
}
func (e *Editor) insertText(y, x int, text string) (int, int) {
	if y < 0 {
		y = 0
	} else if y > len(e.lines) {
		y = len(e.lines)
	}

	if x < 0 {
		x = 0
	} else if x > len(e.lines[y]) {
		x = len(e.lines[y])
	}

	e.lines[y] = e.lines[y][:x] + text + e.lines[y][x:]
	return y, x
}
func (e *Editor) insert(y, x int, text string) {
	e.modified[e.path] = true

	y, x = e.insertText(y, x, text)
	a := Action{
		location: Location{
			line: y,
			col:  x,
		},
		actionType: INSERT,
		text:       text,
		amount:     len(text),
	}
	e.transactions.addAction(a)
}
func (e *Editor) undoTransaction() {
	before := time.Now()
	defer e.debugLog("undo took:", time.Since(before))

	ok, ta := e.transactions.pop()

	if !ok {
		return
	}

	e.debugLog("transactions", len(e.transactions.transactions))

	for _, action := range ta.actions {
		switch action.actionType {
		case DELETE_LINE:
			lines := strings.Split(action.text, "\n")
			e.addLines(action.location.line, lines)
		case DELETE:
			e.insertText(action.location.line, action.location.col, action.text)
		case INSERT:
			e.removeText(action.location.line, action.location.col+action.amount, action.amount)
		}
	}
	e.moveYto(ta.location.line)
	e.moveXto(ta.location.col)
}
func (e *Editor) redoTransaction() {
	before := time.Now()
	defer e.debugLog("redo took:", time.Since(before))

	ok, ta := e.transactions.redoPop()

	if !ok {
		return
	}

	e.debugLog("transactions", len(e.transactions.transactions))

	for _, action := range utils.Reverse(ta.actions) {
		e.debugLog(action.actionType)
		switch action.actionType {
		case DELETE_LINE:
			lines := strings.Split(action.text, "\n")
			e.deleteLinesText(action.location.line, len(lines))
		case DELETE:
			e.removeText(action.location.line, action.location.col+len(action.text), len(action.text))
		case INSERT:
			e.insertText(action.location.line, action.location.col, action.text)
		}
	}
	e.moveYto(ta.location.line)
	e.moveXto(ta.location.col)
}
func (e *Editor) addLines(y int, lines []string) {
	e.debugLog("lines:", len(e.lines))

	// e.lines = slices.Insert(e.lines, y, lines...)
	newList := make([]string, len(e.lines)+len(lines))
	copy(newList, e.lines[:y])
	for i := 0; i < len(lines); i++ {
		newList[y+i] = lines[i]
	}
	rest := e.lines[y:]
	for i := 0; i < len(rest); i++ {
		newList[y+len(lines)+i] = rest[i]
	}

	e.lines = newList
}
func (e *Editor) deleteLinesText(y, num int) (text string) {
	e.modified[e.path] = true
	if len(e.lines) == 1 {
		text = e.lines[y]
		e.lines[y] = ""
	} else {
		deletedLines := e.lines[y:utils.Min(y+num, len(e.lines))]

		text = strings.Join(deletedLines, "\n")
		e.lines = append(e.lines[:y], e.lines[utils.Min(y+num, len(e.lines)):]...)
	}
	return
	// e.clampX()
}
func (e *Editor) deleteLines(y, num int) {
	if len(e.lines) <= 0 {
		return
	}

	colPos := len(e.lines[y])

	text := e.deleteLinesText(y, num)

	ta := Action{
		location: Location{
			col:  colPos,
			line: y,
		},
		actionType: DELETE_LINE,
		text:       text,
	}

	e.transactions.addAction(ta)
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
	if e.path != "" && e.modified[e.path] {
		if _, ok := e.tempFilePaths[e.path]; !ok {
			tempFile, err := os.CreateTemp("", e.openPathsToNames[e.path])
			if err != nil {
				return err
			}
			e.tempFilePaths[e.path] = tempFile.Name()

		}

		data := []byte(strings.Join(e.lines, "\n"))
		err := os.WriteFile(e.tempFilePaths[e.path], data, 0666)
		if err != nil {
			return err
		}

	}
	e.tempFilePos[e.path] = Location{col: e.x, line: e.y}
	filePath = strings.ToLower(filePath)
	e.path = filePath

	var lines []byte
	var err error
	if modified, ok := e.modified[e.path]; ok && modified {
		lines, err = os.ReadFile(e.tempFilePaths[e.path])
		if err != nil {
			return err
		}
	} else {
		lines, err = os.ReadFile(filePath)
		e.modified[e.path] = false
		if err != nil {
			return err
		}
	}

	fileExtension := filepath.Ext(filePath)
	if fileExtension != "" {
		fileExtension = strings.ReplaceAll(fileExtension, ".", "")
		err = e.lexer.SetHighlighting(fileExtension)
		if err != nil {
			e.debugLog(err)
		}
	}

	e.selectedXStart, e.selectedYStart, e.selectedXEnd, e.selectedYEnd = 0, 0, 0, 0
	e.inlinePosition = 0

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

	if loc, ok := e.tempFilePos[e.path]; ok {
		e.moveYto(loc.line)
		e.moveXto(loc.col)
		e.debugLog("loc:", loc.col, loc.line)

	} else {
		e.moveXto(0)
		e.moveYto(0)
	}

	if _, ok := e.openPathsToNames[filePath]; !ok {
		filename := filepath.Base(filePath)
		e.openPathsToNames[filePath] = filename
		e.openedFiles = append(e.openedFiles, filePath)
		e.current = len(e.openedFiles) - 1
	} else {
		e.current = utils.Index(e.openedFiles, filePath)
	}

	e.draw()

	return nil
}
func (e *Editor) moveY(delta int) {
	e.y = utils.Min(utils.Max(e.y+delta, 0), len(e.lines)-1)
	e.debugLog(e.y, len(e.lines))
	e.clampX()

	if e.y-e.printLinesIndex > e.maxY-e.config.TabWidth {
		e.printLinesIndex = e.y - e.maxY + e.config.TabWidth
	} else if e.y-e.config.TabWidth < e.printLinesIndex {
		e.printLinesIndex = utils.Max(e.y-e.config.TabWidth, 0)
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
	} else if delta < 0 {
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
	e.moveX(x - e.x)
}
func (e *Editor) moveYto(y int) {
	e.moveY(y - e.y)
}
func (e *Editor) getTokenIndexByX(tokens []Token, x int) int {
	index := -1
	for i, token := range tokens {
		//e.debugLog("X:", x, "token:", token.location.col, "tokenSize:", token.location.col+token.Length())
		if token.location.col <= x && token.location.col+token.Length() >= x {
			index = i
			// e.debugLog("found")
			break
		}
	}
	//e.debugLog("returning", index)
	//e.debugLog("--------------------")
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
		//e.debugLog("next: ", nextTonken.location.col)
		e.moveX(nextTonken.location.col + nextTonken.Length() - e.x)
	} else {
		e.moveX(tonken.location.col + tonken.Length() - e.x)
	}
}
func (e *Editor) find(text string) (int, int) {
	text = strings.ToLower(text)
	for lineNr, line := range e.lines[e.y:] {
		line = strings.ToLower(line)
		if index := strings.Index(line, text); index != -1 && lineNr != 0 {
			return e.y + lineNr, index
		}
	}
	for lineNr, line := range e.lines[:e.y+1] {
		line = strings.ToLower(line)
		if index := strings.Index(line, text); index != -1 {
			return lineNr, index
		}
	}

	return -1, -1
}
func (e *Editor) Run() error {
	for {
		key := e.stdscr.GetChar()

		//before := time.Now()
		e.debugLog(key, gc.KeyString(key))

		updateLengthIndex := true
		resetSelected := true
		currentLine := e.lines[e.y]

		beforeY, beforeX := e.y, e.x

		// TODO: Make CTRL and Shift bools instead, how to do release tho?
		switch key {
		case gc.KEY_ESC:
			anyUnsaved := false
			for _, modified := range e.modified {
				if modified {
					anyUnsaved = true
					break
				}
			}
			if anyUnsaved {
				str := e.miniWindow.run(true, "unsaved, are you sure? (y/n)")
				if strings.ToLower(str) == "y" {
					return nil
				}
				break
			}

			return nil
		case 559, 563: // ALT + Right
			next := e.current + 1
			if next >= len(e.openedFiles) {
				next = 0
			}
			err := e.Load(e.openedFiles[next])
			if err != nil {
				e.debugLog(err)
			}

		case 544, 548: // ALT + Left
			next := e.current - 1
			if next < 0 {
				next = len(e.openedFiles) - 1
			}
			err := e.Load(e.openedFiles[next])
			if err != nil {
				e.debugLog(err)
			}

		case 562, 566: // CTRL + Shift + Right
			e.ctrlMoveRight()
			resetSelected = false
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y
		case 561, 565: // CTRL + Right
			e.ctrlMoveRight()
		case 526, 530: // CTRL + Down
			e.printLinesIndex = utils.Min(e.printLinesIndex+1, len(e.lines))
			resetSelected = false
		case 547, 551: // CTRL + Shift + Left
			e.ctrlMoveLeft()
			resetSelected = false
			e.selectedXEnd = e.x
			e.selectedYEnd = e.y
		case 546, 550: // CTRL + Left
			e.ctrlMoveLeft()
		case 567, 571: // CTRL + Up
			e.printLinesIndex = utils.Max(e.printLinesIndex-1, 0)
			resetSelected = false
		case 1: // CTRL + A
			e.selectedYStart = 0
			e.selectedXStart = 0
			e.selectedXEnd = len(e.lines[len(e.lines)-1])
			e.selectedYEnd = len(e.lines) - 1

			e.moveYto(e.selectedYEnd)
			e.moveXto(e.selectedXEnd)
			//e.x = e.selectedXEnd
			//e.y = e.selectedYEnd
			resetSelected = false
		case 3: // CTRL + C
			text := e.selected
			if e.selected == "" {
				text = "\n" + currentLine
			}
			err := clipboard.WriteAll(text)
			if err != nil {
				panic(err)
			}
		case 4: // CTRL + D
			e.debugLog("Before y, len", e.y, len(e.lines))
			e.deleteLines(e.y, 1)
			e.moveY(0)
			e.debugLog("After y, len", e.y, len(e.lines))
		case 6: // CTRL + F
			for {
				str := e.miniWindow.run(false, "find")
				if str == "" {
					break
				}

				y, x := e.find(str)
				if y == -1 || x == -1 {
					continue
				}

				e.debugLog("y, x", y, x)
				e.moveYto(y)
				e.moveXto(x)

				e.debugLog("after move y, x", e.y, e.x)

				e.debugLog("x is:", e.x)
				resetSelected = false
				e.selectedXStart = e.x
				e.selectedYStart = e.y
				e.selectedXEnd = e.x + len(str)
				e.selectedYEnd = e.y

				e.draw()
			}
		case 7: // CTRL + G
			str := e.miniWindow.run(true, "goto")
			lineNr, err := strconv.Atoi(str)
			if err != nil {
				break
			}

			if lineNr == -1 {
				lineNr = len(e.lines)
			}

			e.moveXto(0)
			e.inlinePosition = 0
			e.moveYto(lineNr - 1)
		case 15: // CTRL + O
			currentPath := "."
			for {
				files, err := os.ReadDir(currentPath)
				if err != nil {
					e.debugLog(err)
					break
				}

				directoryNames := make([]string, 0)
				fileNames := make([]string, 0)
				for _, file := range files {
					fileInfo, err := os.Stat(filepath.Join(currentPath, file.Name()))
					if err != nil {
						e.debugLog("failed to get stats for file/dir:", err)
						continue
					}

					if fileInfo.IsDir() {
						directoryNames = append(directoryNames, file.Name())
					} else {
						fileNames = append(fileNames, file.Name())
					}
				}

				sort.Strings(directoryNames)
				sort.Strings(fileNames)

				menuItems := make([]MenuItem, 0)
				for _, name := range directoryNames {
					menuItems = append(menuItems, MenuItem{label: name, color: e.config.FolderColor.Color})
				}
				for _, name := range fileNames {
					menuItems = append(menuItems, MenuItem{label: name, color: e.lexer.config.Default.Color})
				}
				e.draw()
				selected, err := e.menuWindow.run(menuItems, currentPath)
				if err != nil {
					e.debugLog(err)
					break
				}
				if selected == "" {
					if currentPath == "." {
						break
					}
					currentPath = filepath.Dir(currentPath)
				}

				currentPath = filepath.Join(currentPath, selected)

				fileInfo, err := os.Stat(currentPath)
				if err != nil {
					e.debugLog(err)
					break
				}

				if fileInfo.IsDir() {
					continue
				}

				err = e.Load(currentPath)
				if err != nil {
					e.debugLog(err)
				}
				break
			}
		case 17: // CTRL + Q Used for testing for now
			e.popupWindow.pop("This is a very long message that is obviously too long")
		case 18: // CTRL + R
			for {
				str1 := e.miniWindow.run(false, "replace(find)")
				if str1 == "" {
					break
				}

				y, x := e.find(str1)
				if y == -1 || x == -1 {
					continue
				}

				e.moveYto(y)
				e.moveXto(x)

				resetSelected = false
				e.selectedXStart = e.x
				e.selectedYStart = e.y
				e.selectedXEnd = e.x + len(str1)
				e.selectedYEnd = e.y

				e.draw()

				str2 := e.miniWindow.run(false, "replace(overwrite)")
				if str2 == "" {
					break
				}

				e.removeSelection()
				e.insert(e.selectedYStart, e.selectedXStart, str2)
				e.selectedXEnd = e.x + len(str2)

				e.draw()
				e.transactions.submit(e.y, e.x)
			}
		case 19: // CTRL + S
			err := e.Save(e.path)
			if err != nil {
				log.Println(err)
				e.popupWindow.pop("Failed to save!")
			} else {
				e.drawHeader()
				e.popupWindow.pop("Saved!")
			}
		case 20: // CTRL + T
			//filenames := make([]MenuItem, len(e.recentToPaths)-1)
			//i := 0
			//for filename, path := range e.recentToPaths {
			//	if path == e.path {
			//		continue
			//	}
			//
			//	filenames[i] = MenuItem{label: filename, color: e.lexer.config.Default.Color}
			//	i++
			//}
			//
			//selected, err := e.menuWindow.run(filenames, "Recent")
			//if err != nil {
			//	e.debugLog(err)
			//	break
			//}
			//
			//if selected != "" && e.recentToPaths[selected] != e.path {
			//	err = e.Load(e.recentToPaths[selected])
			//	if err != nil {
			//		e.debugLog(err)
			//
			//	}
			//	continue
			//}

		case 26: // CTRL + Z
			e.undoTransaction()
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
		case 25: // CTRL + Y
			e.redoTransaction()
		case 336: // Shift+Down
			e.moveY(1)
			updateLengthIndex = false
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x

			// log.Println(e.selectedYEnd, e.selectedYEnd)
		case 337: // Shift+Up
			e.moveY(-1)
			updateLengthIndex = false
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case 393: // Shift+Left
			e.moveX(-1)
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case 402: // Shift+Right
			e.moveX(1)
			resetSelected = false
			e.selectedYEnd = e.y
			e.selectedXEnd = e.x
		case 531, 535: // CTRL+END
			e.moveY(len(e.lines) - e.y)
			updateLengthIndex = false
		case 536, 540: // CTRL+Home
			e.moveY(-e.y)
			updateLengthIndex = false
		case gc.KEY_F5:
			command := e.miniWindow.run(false, "run")
			if command == "" {
				break
			}

			commandList := strings.Split(command, " ")
			cmdName := commandList[0]
			var cmd *exec.Cmd
			if len(commandList) > 1 {
				cmd = exec.Command(cmdName, command[1:])
			} else {
				cmd = exec.Command(cmdName)
			}

			output, err := cmd.CombinedOutput()
			if err != nil {
				e.debugLog("1")
				e.debugLog(err)
				break
			}
			e.debugLog(string(output))

			// e.Run()
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
			if e.selected != "" {
				e.removeSelection()
			}

			newLine := e.lines[e.y][:e.x]
			e.lines[e.y] = e.lines[e.y][e.x:]

			before := make([]string, len(e.lines[:e.y]))
			copy(before, e.lines[:e.y])

			before = append(before, newLine)

			rest := make([]string, len(e.lines[e.y:]))
			copy(rest, e.lines[e.y:])

			e.lines = append(before, rest...)

			e.moveY(1)
			e.moveXto(0)
		case gc.KEY_TAB:
			if e.selected != "" {
				e.removeSelection()
			}

			e.lines[e.y] = e.lines[e.y][:e.x] + "\t" + e.lines[e.y][e.x:]
			e.moveX(1)
		case gc.KEY_SEND:
			e.moveXto(len(e.lines[e.y]))
			resetSelected = false
			e.selectedXEnd = e.x
		case gc.KEY_END:
			e.moveXto(len(e.lines[e.y]))
		case gc.KEY_SHOME:
			e.moveXto(0)
			resetSelected = false
			e.selectedXEnd = e.x
		case gc.KEY_HOME:
			e.moveXto(0)
		case gc.KEY_BACKSPACE:
			if e.selected != "" {
				e.removeSelection()
				break
			}

			x := e.x
			y := e.y
			e.moveX(-1)
			e.remove(y, x, 1)
		default:
			chr := gc.KeyString(key)
			if len(chr) > 1 {
				continue
			}

			if e.selected != "" {
				e.removeSelection()
			}

			// e.lines[e.y] = e.lines[e.y][:e.x] + chr + e.lines[e.y][e.x:]
			e.insert(e.y, e.x, chr)
			e.moveX(1)
			// e.x++
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
		//e.debugLog("run took:", time.Since(before))
		e.draw()
		e.transactions.submit(beforeY, beforeX)
	}
}
func (e *Editor) Save(filepath string) error {
	e.modified[e.path] = false
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
	if len(os.Args) >= 3 && os.Args[2] == "--debug" {
		DEBUG_MODE = true
	}

	InitHomeFolder()

	err := EnsureGimFolderExists()
	if err != nil {
		panic(err)
	}

	homedir := getHomePath()
	f, err := os.Create(JoinPath(homedir, "logs.txt"))
	if err != nil {
		panic(err)
	}

	log.SetOutput(f)
	defer f.Close()

	e := &Editor{}
	e.Init()
	defer e.End()

	if path != "" {
		_ = e.Load(path)
	}

	err = e.Run()
	if err != nil {
		panic(err)
	}
}
