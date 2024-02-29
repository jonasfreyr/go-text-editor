package utils

import "strconv"

func Contains[T comparable](arr []T, el T) bool {
	for _, element := range arr {
		if element == el {
			return true
		}
	}
	return false
}

func ColorToString(color [3]int) string {
	return strconv.Itoa(color[0]) + strconv.Itoa(color[1]) + strconv.Itoa(color[2])
}
