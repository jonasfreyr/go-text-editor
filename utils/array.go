package utils

import "strconv"

func Contains[T comparable](arr []T, el T) bool {
	return Index(arr, el) != -1
}

func Index[T comparable](arr []T, el T) int {
	for i, element := range arr {
		if element == el {
			return i
		}
	}
	return -1
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

type Queue []int

func (q *Queue) Enqueue(item int) {
	*q = append(*q, item)
}

func (q *Queue) Dequeue() int {
	item := (*q)[0]
	*q = (*q)[1:]
	return item
}

func (q *Queue) IsEmpty() bool {
	return len(*q) == 0
}
