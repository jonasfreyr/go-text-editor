package main

import (
	"github.com/jonasfreyr/playground/utils"
	gc "github.com/rthornton128/goncurses"
	"log"
)

var colorIndex = 1
var colorMap map[string]int

var HAS_COLOR = false

func InitColor() error {
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

	HAS_COLOR = true
	colorMap = make(map[string]int)
	return nil
}

func setColor(color [3]int) error {
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
	colorMap[key] = colorIndex

	colorIndex++

	return nil
}
func DisableColor(scr *gc.Window, color [3]int) {
	if !HAS_COLOR {
		return
	}

	key := utils.ColorToString(color)
	colorIndex, ok := colorMap[key]
	if !ok {
		log.Println("invalid color key:", color)
		return
	}

	err := scr.ColorOff(int16(colorIndex))
	if err != nil {
		log.Println(err)
	}
}
func EnableColor(scr *gc.Window, color [3]int) {
	if !HAS_COLOR {
		return
	}

	key := utils.ColorToString(color)
	colorIndex, ok := colorMap[key]
	if !ok {
		err := setColor(color)
		if err != nil {
			log.Println(err)
			return
		}
		colorIndex = colorMap[key]
	}

	err := scr.ColorOn(int16(colorIndex))
	if err != nil {
		log.Println(err)
	}
}
