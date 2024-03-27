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

func Reverse[K any](arr []K) []K {
	newArray := make([]K, len(arr))
	for i, j := 0, len(arr)-1; i < len(arr); i, j = i+1, j-1 {
		newArray[i] = arr[j]
	}
	return newArray
}
