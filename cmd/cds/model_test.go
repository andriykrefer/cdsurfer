package main

import (
	"fmt"
	"testing"

	"github.com/andriykrefer/cdsurfer/config"
)

func TestCalculateColsAndRows(t *testing.T) {
	config.FILES_SEPARATOR_SZ = 2
	m := Model{
		width: 20,
		items: []Item{
			{name: "1"},
			{name: "12"},
			{name: "123/"},
		},
	}

	m.calculateColsAndRows()
	fmt.Println(m.rows)
	fmt.Println(m.width)
	fmt.Println(m.cols)
	fmt.Println(m.colSize)
	fmt.Println("")

	if m.rows != 1 {
		t.Fail()
	}
	if m.width != 20 {
		t.Fail()
	}
	if m.cols != 3 {
		t.Fail()
	}
	if m.colSize != 6 {
		t.Fail()
	}

	m = Model{
		width: 20,
		items: []Item{
			{name: "1234567"},
			{name: "1234567"},
			{name: "1234567"},
			{name: "1234567"},
			{name: "1234567"},
		},
	}

	m.calculateColsAndRows()
	fmt.Println(m.rows)
	fmt.Println(m.width)
	fmt.Println(m.cols)
	fmt.Println(m.colSize)
	fmt.Println("")

	if m.rows != 3 {
		t.Fail()
	}
	if m.width != 20 {
		t.Fail()
	}
	if m.cols != 2 {
		t.Fail()
	}
	if m.colSize != 10 {
		t.Fail()
	}

}
