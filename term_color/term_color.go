package term_color

import (
	"github.com/gookit/color"
)

var RESET = "\033[0m"

func Violet(s string, isBg bool) string {
	return color.C256(93, isBg).Sprint(s)
}

func Orange(s string, isBg bool) string {
	return color.C256(208, isBg).Sprint(s)
}

func BlueBold(s string, isBg bool) string {
	c := color.S256(33).SetOpts(color.Opts{color.OpBold})
	return c.Sprint(s)
}

func Blue(s string, isBg bool) string {
	return color.C256(33, isBg).Sprint(s)
}

func Green(s string, isBg bool) string {
	return color.C256(34, isBg).Sprint(s)
}

func Gray(s string, isBg bool) string {
	return color.C256(245, isBg).Sprint(s)
}

func ClearScreen() string {
	// Clear screen + cursor home
	return "\033[2J" + "\033[H"
}
