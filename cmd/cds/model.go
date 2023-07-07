package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andriykrefer/cds/config"
	"github.com/andriykrefer/cds/exp"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type stateEnum int

const (
	stateList   stateEnum = 0
	stateSearch stateEnum = 1
)

type Model struct {
	path      string
	cursorIx  int
	rowOffset int
	colSize   int
	cols      int
	rows      int
	items     []Item
	width     int
	height    int
	state     stateEnum
	hasFit    bool
}

type Item struct {
	name       string
	fileInfo   os.FileInfo
	isSelected bool
}

func (thiss *Model) Init() tea.Cmd {
	return nil
}

// https://github.com/charmbracelet/bubbletea/blob/master/key.go
var (
	keyKill     = key.NewBinding(key.WithKeys("ctrl+c"))
	keyQuit     = key.NewBinding(key.WithKeys("esc"))
	keyUp       = key.NewBinding(key.WithKeys("up"))
	keyDown     = key.NewBinding(key.WithKeys("down"))
	keyLeft     = key.NewBinding(key.WithKeys("left"))
	keyRight    = key.NewBinding(key.WithKeys("right"))
	keyEnter    = key.NewBinding(key.WithKeys("enter"))
	keySpace    = key.NewBinding(key.WithKeys(" "))
	keyBack     = key.NewBinding(key.WithKeys("backspace"))
	keyPageUp   = key.NewBinding(key.WithKeys("pgup"))
	keyPageDown = key.NewBinding(key.WithKeys("pgdown"))
	keyHome     = key.NewBinding(key.WithKeys("home"))
	keyEnd      = key.NewBinding(key.WithKeys("end"))
)

func (thiss *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		thiss.width = msg.Width
		thiss.height = msg.Height
		thiss.calculateColsAndRows()
		return thiss, nil
	case tea.KeyMsg:
		if thiss.state == stateList {
			return thiss.updateStateList(msg)
		}
	}
	return thiss, nil
}

func (thiss *Model) updateStateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyKill):
		_, _ = fmt.Fprintln(os.Stderr)
		return thiss, tea.Quit

	case key.Matches(msg, keyQuit):
		_, _ = fmt.Fprintln(os.Stderr)
		fmt.Println(thiss.path)
		return thiss, tea.Quit

	case key.Matches(msg, keyLeft):
		thiss.cursorAdd(-1)
		if !thiss.isCursorDisplayed() {
			thiss.addRowOffset(-1)
		}
		return thiss, nil

	case key.Matches(msg, keyRight):
		thiss.cursorAdd(1)
		if !thiss.isCursorDisplayed() {
			thiss.addRowOffset(1)
		}
		return thiss, nil

	case key.Matches(msg, keyUp):
		thiss.cursorAdd(-thiss.cols)
		if !thiss.isCursorDisplayed() {
			thiss.addRowOffset(-1)
		}
		return thiss, nil

	case key.Matches(msg, keyDown):
		thiss.cursorAdd(thiss.cols)
		if !thiss.isCursorDisplayed() {
			thiss.addRowOffset(1)
		}
		return thiss, nil

	case key.Matches(msg, keyPageDown):
		thiss.cursorAdd(thiss.cols * thiss.rowsDisplayed())
		thiss.rowOffset = thiss.cursorRowIx()
		return thiss, nil

	case key.Matches(msg, keyPageUp):
		thiss.cursorAdd(-thiss.cols * thiss.rowsDisplayed())
		thiss.rowOffset = thiss.cursorRowIx()
		return thiss, nil

	case key.Matches(msg, keyHome):
		thiss.cursorIx = 0
		thiss.rowOffset = 0
		return thiss, nil

	case key.Matches(msg, keyEnd):
		thiss.cursorIx = len(thiss.items) - 1
		thiss.rowOffset = thiss.cursorRowIx()
		return thiss, nil

	case key.Matches(msg, keyEnter):
		thiss.cursorEnter()
		return thiss, nil

	case key.Matches(msg, keySpace):
		thiss.toggleSelection()
		return thiss, nil
	}
	return thiss, nil
}

func (thiss *Model) View() string {
	out := ""
	totalAbsRows := exp.TryFallback(func() int { return ((len(thiss.items) - 1) / thiss.cols) + 1 }, 0)
	for ix, item := range thiss.items {
		col := exp.TryFallback(func() int { return ix % thiss.cols }, 0)
		row := exp.TryFallback(func() int { return ix / thiss.cols }, 0)
		if row < thiss.rowOffset {
			continue
		}
		relRow := max(row-thiss.rowOffset, 0)

		if col == 0 {
			isLastLine := row >= (thiss.rowOffset + thiss.rowsDisplayed())
			willFit := row == (totalAbsRows - 1)
			if isLastLine && !willFit {
				out += lipgloss.NewStyle().
					Foreground(lipgloss.Color(config.FG_MORE)).
					Render("\n--More--")
				break
			} else if relRow != 0 {
				out += "\n"
			}
		}
		out += thiss.renderItem(item, thiss.cursorIx == ix)
	}
	ret := "=== HEADER ===\n" + out
	ret = lipgloss.NewStyle().
		Height(thiss.height - 3).
		Render(ret)
	ret += "\n" + renderFooter()
	return ret
}

func renderFooter() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(config.FG_DISCREET)).Render("" +
		"space: Select    alt+c: Copy    alt+x: Cut    alt+v: Paste\n" +
		"alpha: Search    alt+h: Help")
}

func (thiss *Model) Ls() {
	files, err := os.ReadDir(thiss.path)
	if err != nil {
		panic(err)
	}
	thiss.items = []Item{{name: "../"}}
	for _, f := range files {
		info, _ := f.Info()
		name := info.Name()
		if info.IsDir() {
			name += "/"
		}
		thiss.items = append(thiss.items, Item{
			name:     name,
			fileInfo: info,
		})
	}

	if config.LIST_FOLDERS_FIRST {
		thiss.items = sortItemsFoldersFirst(thiss.items)
	}
}

func sortItemsFoldersFirst(items []Item) []Item {
	folders := []Item{}
	files := []Item{}
	other := []Item{}
	for _, i := range items {
		if i.fileInfo == nil {
			other = append(other, i)
			continue
		}
		if i.fileInfo.IsDir() {
			folders = append(folders, i)
			continue
		}

		files = append(files, i)
	}

	ret := append(other, folders...)
	ret = append(ret, files...)
	return ret
}

func (thiss *Model) renderItem(item Item, isFocused bool) string {
	style := lipgloss.NewStyle()
	// if thiss.rows == 1 {
	// style = style.MarginRight(config.FILES_SEPARATOR_SZ)
	// } else {
	style = style.Width(thiss.colSize)
	// }

	if item.fileInfo == nil {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_FOLDER))
	} else if item.fileInfo.IsDir() {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_FOLDER))
	} else {
		//
	}

	if item.isSelected {
		style = style.Background(lipgloss.Color(config.BG_SELECTED))
	}

	if isFocused {
		// style = style.Reverse(true)
		style = style.Background(lipgloss.Color(config.BG_CURSOR))
	}
	return style.Render(item.name)
}

func (thiss *Model) calculateColsAndRows() {
	// itemsNames := []string{}
	// for _, i := range thiss.items {
	// 	itemsNames = append(itemsNames, i.name)
	// }
	// doesFitInOneCol := len(
	// 	strings.Join(
	// 		itemsNames,
	// 		strings.Repeat(".", config.FILES_SEPARATOR_SZ),
	// 	),
	// ) < thiss.width
	// if doesFitInOneCol {
	// 	thiss.cols = 0
	// 	thiss.rows = 1
	// 	return
	// }

	maxColSize := thiss.maxItemLength() + config.FILES_SEPARATOR_SZ
	if maxColSize >= thiss.width {
		thiss.cols = 1
		thiss.colSize = thiss.width
		thiss.rows = len(thiss.items)
		return
	}
	thiss.cols = thiss.width / maxColSize
	thiss.colSize = thiss.width / thiss.cols
	thiss.rows = len(thiss.items) / thiss.cols
	if len(thiss.items)%thiss.cols > 0 {
		thiss.rows += 1
	}
}

func (thiss *Model) maxItemLength() int {
	max := 0
	for _, i := range thiss.items {
		if len(i.name) > max {
			max = len(i.name)
		}
	}
	return max
}

func (thiss *Model) cursorAdd(val int) {
	thiss.cursorIx = minMax(thiss.cursorIx+val, 0, len(thiss.items)-1)
}

// func (thiss *Model) cursorLeft() {
// 	thiss.cursorIx -= 1
// 	if thiss.cursorIx < 0 {
// 		thiss.cursorIx = 0
// 	}
// }

// func (thiss *Model) cursorRight() {
// 	thiss.cursorIx += 1
// 	if thiss.cursorIx > len(thiss.items)-1 {
// 		thiss.cursorIx = len(thiss.items) - 1
// 	}
// }

// func (thiss *Model) cursorUp() {
// 	thiss.cursorIx = max(thiss.cursorIx-thiss.cols, 0)
// }

// func (thiss *Model) cursorDown() {
// 	thiss.cursorIx = min(thiss.cursorIx+thiss.cols, len(thiss.items)-1)
// }

func (thiss *Model) isCursorDisplayed() bool {
	cursor := thiss.cursorRowIx()
	return (cursor < (thiss.rowOffset + thiss.rowsDisplayed())) && (cursor >= thiss.rowOffset)
}

func (thiss *Model) CurrentItem() Item {
	return thiss.items[thiss.cursorIx]
}

func (thiss *Model) cursorRowIx() int {
	return (thiss.cursorIx / thiss.cols)
}

func (thiss *Model) rowsDisplayed() int {
	// if thiss.hasFit {
	// 	return max(thiss.height-4, 1)
	// }
	return max(thiss.height-5, 1)
}

func (thiss *Model) addRowOffset(val int) {
	thiss.rowOffset = minMax(thiss.rowOffset+val, 0, thiss.rows-1)
}

func (thiss *Model) cursorEnter() {
	curItem := thiss.CurrentItem()
	if curItem.fileInfo != nil && !curItem.fileInfo.IsDir() {
		return
	}

	thiss.path = filepath.Clean(
		filepath.Join(thiss.path, thiss.CurrentItem().name),
	)
	thiss.cursorIx = 0
	thiss.Ls()
	thiss.calculateColsAndRows()
}

func (thiss *Model) toggleSelection() {
	if thiss.items[thiss.cursorIx].fileInfo == nil {
		return
	}
	thiss.items[thiss.cursorIx].isSelected = !thiss.items[thiss.cursorIx].isSelected
}
