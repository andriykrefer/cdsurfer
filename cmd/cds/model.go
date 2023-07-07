package main

import (
	"fmt"
	"os"
	"os/user"
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
	path        string
	cursorIx    int
	rowOffset   int
	colSize     int
	cols        int
	rows        int
	items       []Item
	width       int
	height      int
	state       stateEnum
	username    string
	showDetails bool
}

type Item struct {
	name       string
	fileInfo   os.FileInfo
	isSelected bool
}

func (thiss *Model) Init() tea.Cmd {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	currentUser, err := user.Current()
	if err != nil {
		panic(err.Error())
	}
	username := currentUser.Username

	path = "/"
	thiss.path = path
	thiss.width = 80
	thiss.height = 10
	thiss.username = username
	thiss.Ls()
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
	keyCopy     = key.NewBinding(key.WithKeys("C"))
	keyCut      = key.NewBinding(key.WithKeys("alt+x"))
	keyPaste    = key.NewBinding(key.WithKeys("alt+v"))
	keyDetails  = key.NewBinding(key.WithKeys("alt+d"))
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

	case key.Matches(msg, keyCopy):
		thiss.toggleSelection()
		return thiss, nil

	case key.Matches(msg, keyCut):
		thiss.toggleSelection()
		return thiss, nil

	case key.Matches(msg, keyPaste):
		thiss.toggleSelection()
		return thiss, nil

	case key.Matches(msg, keyDetails):
		thiss.toggleDetails()
		return thiss, nil
	}
	return thiss, nil
}

func (thiss *Model) View() string {
	if thiss.state == stateList {
		return thiss.renderList()
	}
	return ""
}

func (thiss *Model) renderList() string {
	if thiss.showDetails {
		listOut := ""
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
					listOut += lipgloss.NewStyle().
						Foreground(lipgloss.Color(config.FG_MORE)).
						Render("\n--More--")
					break
				} else if relRow != 0 {
					listOut += "\n"
				}
			}
			listOut += thiss.renderItemWithDetails(item, thiss.cursorIx == ix)
		}

		return thiss.renderListScreen(thiss.renderHeader(), listOut, renderFooter())
	} else {
		listOut := ""
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
					listOut += lipgloss.NewStyle().
						Foreground(lipgloss.Color(config.FG_MORE)).
						Render("\n--More--")
					break
				} else if relRow != 0 {
					listOut += "\n"
				}
			}
			listOut += thiss.renderItem(item, thiss.cursorIx == ix)
		}

		return thiss.renderListScreen(thiss.renderHeader(), listOut, renderFooter())
	}
}

func (thiss *Model) renderListScreen(header, list, footer string) string {
	ret := header + "\n" + list
	// ret = lipgloss.NewStyle().
	// 	Height(thiss.height - 3).
	// 	Render(ret)
	ret += "\n" + footer
	return ret
}

func (thiss *Model) renderHeader() string {
	return thiss.username + ": " + thiss.path
	// return lipgloss.NewStyle().
	// 	// Bold(true).
	// 	// Underline(true).
	// 	// Background(lipgloss.Color(config.BG_PATH)).
	// 	Foreground(lipgloss.Color(config.FG_PATH)).
	// 	Render(thiss.username+": "+thiss.path) + "\n"
}

func renderFooter() string {
	return ""
	return lipgloss.NewStyle().Foreground(lipgloss.Color(config.FG_DISCREET)).Render("" +
		"space: Select   shift+c: Copy   alt+x: Cut      alt+v: Paste   del: Delete" +
		"\n" +
		"alt+h: Help     esc: Quit       lower: Search   alt+d: Details" +
		"",
	)
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

// mergeFileStyle add colors and bold style to an initial style. Based on file types
func mergeFileStyle(initialStyle lipgloss.Style, item Item, isFocused bool) lipgloss.Style {
	isExecutable := func(item Item) bool {
		if item.fileInfo == nil {
			return false
		}
		perm := item.fileInfo.Mode().String()
		return (perm[3:4] == "x") ||
			(perm[6:7] == "x") ||
			(perm[9:10] == "x")
	}

	isSymlink := func(item Item) bool {
		if item.fileInfo == nil {
			return false
		}
		return (item.fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink)
	}

	style := initialStyle
	if item.fileInfo == nil {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_FOLDER))
	} else if item.fileInfo.IsDir() {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_FOLDER))
	} else if isSymlink(item) {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_SYMLINK))
	} else if isExecutable(item) {
		style = style.
			Bold(true).
			Foreground(lipgloss.Color(config.FG_EXEC))
	}

	if item.isSelected {
		style = style.Background(lipgloss.Color(config.BG_SELECTED))
	}

	if isFocused {
		style = style.Background(lipgloss.Color(config.BG_CURSOR))
	}

	return style
}

func (thiss *Model) renderItem(item Item, isFocused bool) string {
	style := lipgloss.NewStyle().Width(thiss.colSize)
	style = mergeFileStyle(style, item, isFocused)
	return style.Render(item.name)
}

func (thiss *Model) renderItemWithDetails(item Item, isFocused bool) string {
	style := lipgloss.NewStyle().Width(thiss.colSize)
	style = mergeFileStyle(style, item, isFocused)

	details := "drwxrwxr-x 5 andriy andriy 4,0K jul  7 03:29 "
	if item.fileInfo != nil {
		perm := item.fileInfo.Mode().String()
		details = perm[3:4] + "" +
			perm[6:7] + "" +
			perm[9:10]

	}
	return details + "    " + style.Render(item.name)
}

func (thiss *Model) calculateColsAndRows() {

	if thiss.showDetails {
		thiss.cols = 1
		thiss.colSize = thiss.width
		thiss.rows = len(thiss.items)
		return
	}

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

func (thiss *Model) toggleDetails() {
	thiss.showDetails = !thiss.showDetails
	thiss.calculateColsAndRows()
}
