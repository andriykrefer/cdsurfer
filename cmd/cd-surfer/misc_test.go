package main

import (
	"testing"
)

func TestMinMax(t *testing.T) {
	if minMax(10, 1, 20) != 10 {
		t.Fail()
	}
	if minMax(0, 1, 20) != 1 {
		t.Fail()
	}
	if minMax(21, 0, 20) != 20 {
		t.Fail()
	}
}
