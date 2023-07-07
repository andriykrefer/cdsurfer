package main

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minMax(x, xMin, xMax int) int {
	return min(xMax, max(xMin, x))
}
