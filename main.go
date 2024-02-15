package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
)

type highlightingConfig struct {
	Literals      []string `json:"literals"`
	BuiltIns      []string `json:"built_ins"`
	Types         []string `json:"types"`
	Keywords      []string `json:"keywords"`
	Comment       []string `json:"comment"`
	LiteralsColor []int    `json:"literals-color"`
	BuiltInsColor []int    `json:"built_ins-color"`
	TypesColor    []int    `json:"types-color"`
	KeywordsColor []int    `json:"keywords-color"`
	CommentColor  []int    `json:"comment-color"`
}

type Editor struct {
	text   []string
	stdscr *gc.Window

	maxX, maxY int

	x, y int

	currLengthIndex int

	printLineStartIndex int
	printLinesIndex     int

	highlightWords map[string]int
}

func toTokens(line string) []string {
	newLine := make([]string, 0)
	currentToken := ""
	for _, c := range line {
		char := string(c)

		if char == " " {
			if currentToken != "" {
				newLine = append(newLine, currentToken)
			}
			newLine = append(newLine, char)
			currentToken = ""
			continue
		}

		currentToken += char
	}

	if currentToken != "" {
		newLine = append(newLine, currentToken)
	}
	return newLine
}

func (e *Editor) enableIfColor(word string) int {
	if color, ok := e.highlightWords[word]; ok {
		e.stdscr.ColorOn(int16(color))
		return color
	}
	return 0
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

		tokens := toTokens(line)

		// line = line[e.printLineStartIndex:]
		// if len(line) > e.maxX-1 {
		//    line = line[:e.maxX-1]
		// }

		currentPrintIndex := 0
		for _, token := range tokens {
			color := e.enableIfColor(token)

			if currentPrintIndex > e.maxX {
				break
			}
			if currentPrintIndex < e.printLineStartIndex && currentPrintIndex+len(token) < e.printLineStartIndex {
				continue
			}

			if currentPrintIndex < e.printLineStartIndex && currentPrintIndex+len(token) > e.printLineStartIndex {
				token = token[e.printLineStartIndex-currentPrintIndex:]
			}

			if currentPrintIndex+len(token) > e.maxX {
				token = token[:e.maxX-1-currentPrintIndex]
			}

			e.stdscr.Print(token)
			currentPrintIndex += len(token)

			if color != 0 {
				e.stdscr.ColorOff((int16)(color))
			}
		}
		e.stdscr.Println()

	}

	e.stdscr.Move(e.y-e.printLinesIndex, e.x-e.printLineStartIndex)
	e.stdscr.Refresh()

	gc.Cursor(1)
}

func (e *Editor) setColor(index int, color []int) error {
	err := gc.InitColor(int16(index), int16(color[0])*4, int16(color[1])*4, int16(color[2]*4))
	if err != nil {
		return err
	}

	err = gc.InitPair(int16(index), int16(index), -1)
	if err != nil {
		return err
	}

	return nil
}

func (e *Editor) setColors() error {
	f, err := os.Open("highlighting.json")
	if err != nil {
		return err
	}
	defer f.Close()

	var config highlightingConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return err
	}

	highlightingTypes := []string{"literals", "built_ins", "types", "keywords", "comment"}

	e.highlightWords = make(map[string]int)

	highlightWords := make(map[string][]string)
	highlightWords["literals"] = config.Literals
	highlightWords["built_ins"] = config.BuiltIns
	highlightWords["types"] = config.Types
	highlightWords["keywords"] = config.Keywords
	highlightWords["comment"] = config.Comment

	for i, colorArray := range [][]int{config.LiteralsColor, config.BuiltInsColor, config.TypesColor, config.KeywordsColor, config.CommentColor} {
		err = e.setColor(i+1, colorArray)
		if err != nil {
			return err
		}
		for _, word := range highlightWords[highlightingTypes[i]] {
			e.highlightWords[word] = i + 1
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
	e.stdscr, err = gc.Init()

	if err != nil {
		log.Fatal("init", err)
	}

	err = e.initColor()
	if err != nil {
		e.End()
		log.Fatal(err)
	}

	gc.Echo(false)
	e.stdscr.Keypad(true)

	e.maxY, e.maxX = e.stdscr.MaxYX()

	e.text = make([]string, 1)
}
func (e *Editor) End() {
	gc.End()
}

// This needs testing when it comes to multi line deletes
func (e *Editor) deleteLines(y, num int) {
	if len(e.text) <= 0 {
		return
	} else if len(e.text) == 1 {
		e.text[y] = ""
	} else {
		e.text = append(e.text[:y], e.text[y+num:]...)
	}
}
func (e *Editor) clampXToLineOrLengthIndex() {
	if e.currLengthIndex > len(e.text[e.y]) {
		e.x = len(e.text[e.y])
	} else {
		e.x = e.currLengthIndex
	}
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

		if chr == "\t" {
			chr = "    "
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
		case 561: // CTRL + Right
			str := e.text[e.y][e.x:]
			i := strings.Index(str, " ")
			if i == -1 {
				e.x = len(e.text[e.y])
			} else if i == 0 {
				e.x++
			} else {
				e.x = e.x + i
			}
		case 526: // CTRL + Down
		case 546: // CTRL + Left
			str := e.text[e.y][:e.x]
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
			e.y = utils.Min(utils.Max(len(e.text)-1, 0), utils.Max(e.y, 0))
			e.clampXToLineOrLengthIndex()
		case 24: // CTRL + X
			text := e.text[e.y] + "\n"
			err := clipboard.WriteAll(text)
			e.deleteLines(e.y, 1)
			if err != nil {
				panic(err)
			}
		case gc.KEY_DOWN:
			if e.y >= len(e.text)-1 {
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

		e.y = utils.Min(utils.Max(len(e.text)-1, 0), e.y)
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

	f, err := os.Create("info.txt")
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
