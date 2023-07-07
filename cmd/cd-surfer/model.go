package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/andriykrefer/cdsurfer/config"
	"github.com/andriykrefer/cdsurfer/exp"
	"github.com/andriykrefer/cdsurfer/term_color"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modeEnum int

const (
	modeList   modeEnum = 0
	modeSearch modeEnum = 1
)

type Model struct {
	// state
	path          string
	cursorIx      int
	rowOffset     int
	colSize       int
	cols          int
	rows          int
	items         []Item // Items on the current view (may be equal to filteredItems or dirItems)
	dirItems      []Item // All dir items
	filteredItems []Item // Filtered Items on search view
	width         int
	height        int
	mode          modeEnum
	username      string
	showDetails   bool
	searchInput   string
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

	thiss.path = path
	thiss.width = 80
	thiss.height = 10
	thiss.username = username
	thiss.showDetails = config.SHOW_DETAILS
	thiss.Ls()
	return nil
}

// https://github.com/charmbracelet/bubbletea/blob/master/key.go
var (
	keyEsc = key.NewBinding(key.WithKeys("esc"))
	// keyQuit          = key.NewBinding(key.WithKeys("esc", "ctrl+c"))
	keyQuit          = key.NewBinding(key.WithKeys("ctrl+c"))
	keyQuitWithoutCd = key.NewBinding(key.WithKeys("alt+q"))
	keyUp            = key.NewBinding(key.WithKeys("up"))
	keyDown          = key.NewBinding(key.WithKeys("down"))
	keyLeft          = key.NewBinding(key.WithKeys("left"))
	keyRight         = key.NewBinding(key.WithKeys("right"))
	keyEnter         = key.NewBinding(key.WithKeys("enter"))
	keySpace         = key.NewBinding(key.WithKeys(" "))
	keyBack          = key.NewBinding(key.WithKeys("backspace"))
	keyParent        = key.NewBinding(key.WithKeys("alt+backspace"))
	keyPageUp        = key.NewBinding(key.WithKeys("pgup"))
	keyPageDown      = key.NewBinding(key.WithKeys("pgdown"))
	keyHome          = key.NewBinding(key.WithKeys("home"))
	keyEnd           = key.NewBinding(key.WithKeys("end"))
	keyCopy          = key.NewBinding(key.WithKeys("C"))
	keyCut           = key.NewBinding(key.WithKeys("alt+x"))
	keyPaste         = key.NewBinding(key.WithKeys("alt+v"))
	keyDetails       = key.NewBinding(key.WithKeys("alt+d"))
	keyClear         = key.NewBinding(key.WithKeys("ctrl+u"))
)

func (thiss *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		thiss.width = msg.Width
		thiss.height = msg.Height
		thiss.calculateColsAndRows()
		return thiss, nil
	case tea.KeyMsg:
		return thiss.updateStateList(msg)
	}
	return thiss, nil
}

func (thiss *Model) updateStateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyQuit):
		fmt.Println(`cd "` + thiss.path + `"`)
		return thiss, tea.Quit

	case key.Matches(msg, keyQuitWithoutCd):
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
		thiss.changeMode(modeList)
		return thiss, nil

	case key.Matches(msg, keyParent) && thiss.mode == modeList:
		thiss.goBack()
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

	case key.Matches(msg, keyPaste) && thiss.mode == modeList:
		thiss.toggleSelection()
		return thiss, nil

	case key.Matches(msg, keyDetails) && thiss.mode == modeList:
		thiss.toggleDetails()
		return thiss, nil

	case key.Matches(msg, keyEsc, keyClear) && thiss.mode == modeSearch:
		thiss.changeMode(modeList)
		return thiss, nil

	case msg.Type == tea.KeyRunes: // Change to modeSearch
		thiss.searchInput += string(msg.Runes)
		thiss.searchFilter(thiss.searchInput)
		thiss.changeMode(modeSearch)

	case key.Matches(msg, keyBack) && thiss.mode == modeSearch:
		thiss.searchInput = thiss.searchInput[:len(thiss.searchInput)-1]
		if thiss.searchInput == "" {
			thiss.changeMode(modeList)
		}
		return thiss, nil
	}
	return thiss, nil
}

func (thiss *Model) View() string {
	if thiss.mode == modeList || thiss.mode == modeSearch {
		return thiss.renderList()
	}
	return ""
}

func (thiss *Model) renderList() string {
	items := thiss.items
	listOut := ""
	totalAbsRows := exp.TryFallback(func() int { return ((len(items) - 1) / thiss.cols) + 1 }, 0)
	for ix, item := range items {
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
				listOut += "\n" + term_color.Violet("--More--", false)
				break
			} else if relRow != 0 {
				listOut += "\n"
			}
		}
		if thiss.showDetails {
			listOut += thiss.renderItemWithDetails(item, thiss.cursorIx == ix)
		} else {
			listOut += thiss.renderItem(item, thiss.cursorIx == ix)
		}
	}

	return thiss.renderListScreen(thiss.renderHeader(), listOut, renderFooter())
}

func (thiss *Model) renderListScreen(header, list, footer string) string {
	ret := header + "\n" + list
	ret += "\n" + footer
	return ret
}

func (thiss *Model) renderHeader() string {
	o := thiss.username + ": " + thiss.path
	if thiss.mode == modeSearch {
		o = strings.TrimSuffix(o, "/") + "/" + term_color.Violet(thiss.searchInput, false)
	}
	return o
}

func renderFooter() string {
	s := "" +
		// "space: Select   shift+c: Copy   alt+x: Cut      alt+v: Paste   del: Delete" +
		// "\n" +
		// "alt+h: Help     esc: Quit       lower: Search   alt+d: Details"
		"esc: Quit   alt+d: Details"

	return term_color.Gray(s, false)
}

func (thiss *Model) Ls() {
	files, err := os.ReadDir(thiss.path)
	if err != nil {
		panic(err)
	}
	addPreviousDir := func() []Item {
		if filepath.Clean(thiss.path) == "/" {
			return []Item{}
		}
		previousDirStat, err := os.Stat(filepath.Clean(filepath.Join(thiss.path, "..")))
		if err != nil {
			panic(err)
		}
		return []Item{{
			name:     "../",
			fileInfo: previousDirStat,
		}}
	}

	thiss.items = addPreviousDir()
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
	thiss.dirItems = thiss.items
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

func addColorByFileType(text string, item Item, isFocused bool) string {
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

	if isFocused {
		text = term_color.Violet(text, true)
	} else if item.isSelected {
		text = term_color.Orange(text, true)
	} else if item.fileInfo == nil {
		text = term_color.Blue(text, false)
	} else if item.fileInfo.IsDir() {
		text = term_color.Blue(text, false)
	} else if isSymlink(item) {
		// text = term_color.Yellow(text)
		// style = style.UnsetBold()
		// style = style.
		// Bold(true).
		// Foreground(lipgloss.Color(config.FG_SYMLINK))
	} else if isExecutable(item) {
		text = term_color.Green(text, false)
	}

	return text
}
func (thiss *Model) renderItem(item Item, isFocused bool) string {
	style := lipgloss.NewStyle().Width(thiss.colSize)
	return style.Render(addColorByFileType(item.name, item, isFocused))
}

func (thiss *Model) renderItemWithDetails(item Item, isFocused bool) string {
	style := lipgloss.NewStyle().Width(thiss.colSize)

	// details := "drwxrwxr-x 5 andriy andriy 4,0K jul  7 03:29 "
	details := item.fileInfo.Mode().String()
	return term_color.Gray(details, false) + "    " + style.Render(addColorByFileType(item.name, item, isFocused))
}

func (thiss *Model) changeMode(mode modeEnum) {
	if mode == modeSearch {
		thiss.mode = modeSearch
		thiss.items = thiss.filteredItems
		thiss.calculateColsAndRows()
	} else if mode == modeList {
		thiss.searchInput = ""
		thiss.mode = modeList
		thiss.items = thiss.dirItems
		thiss.calculateColsAndRows()
	}
}

func (thiss *Model) searchFilter(input string) {
	filtered := []Item{}
	for _, v := range thiss.dirItems {
		var a = strings.ToLower(input)
		var b = strings.ToLower(v.name)
		if strings.Contains(b, a) {
			filtered = append(filtered, v)
		}
	}
	thiss.filteredItems = filtered
}

func (thiss *Model) calculateColsAndRows() {

	if thiss.showDetails || thiss.mode == modeSearch {
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
	thiss.rowOffset = 0
	thiss.Ls()
	thiss.calculateColsAndRows()
}

func (thiss *Model) goBack() {
	thiss.path = filepath.Clean(
		filepath.Join(thiss.path, ".."),
	)
	thiss.cursorIx = 0
	thiss.rowOffset = 0
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
	// Re calculate offset. Put on the center of the screen
	thiss.rowOffset = 0
	thiss.addRowOffset(thiss.cursorRowIx() - (thiss.rowsDisplayed() / 2))
}
