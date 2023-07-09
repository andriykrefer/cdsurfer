package term

import "strings"

func Width(s string, size int) string {
	if len(s) > size {
		return s[:size]
	}
	var paddingSize = size - len(s)
	return s + strings.Repeat(" ", paddingSize)
}
