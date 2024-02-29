package utils

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func MapTo1000(value int) int {
	return int(float64(value) / 255 * 1000)
}
