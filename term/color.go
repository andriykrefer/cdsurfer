package term

import (
	"github.com/gookit/color"
)

var RESET = "\033[0m"

func Violet(s string, isBg bool) string {
	return color.C256(93, isBg).Sprint(s)
}

func Emphasis(s string) string {
	return color.C256(93, false).Sprint(s)
}

func Orange(s string, isBg bool) string {
	return color.C256(208, isBg).Sprint(s)
}

func Yellow(s string, isBg bool) string {
	return color.C256(226, isBg).Sprint(s)
}

func BlueBold(s string, isBg bool) string {
	c := color.S256(39).SetOpts(color.Opts{color.OpBold})
	return c.Sprint(s)
}

func Blue(s string, isBg bool) string {
	return color.C256(75, isBg).Sprint(s)
}

func Cyan(s string, isBg bool) string {
	return color.C256(51, isBg).Sprint(s)
}

func Green(s string, isBg bool) string {
	return color.C256(34, isBg).Sprint(s)
}

func Gray(s string, isBg bool) string {
	return color.C256(245, isBg).Sprint(s)
}

func Red(s string, isBg bool) string {
	return color.C256(1, isBg).Sprint(s)
}

func ClearScreen() string {
	// Clear screen + cursor home
	return "\033[2J" + "\033[H"
}
