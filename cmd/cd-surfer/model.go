package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"github.com/andriykrefer/cdsurfer/config"
	"github.com/andriykrefer/cdsurfer/term"
	"github.com/andriykrefer/exp"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type modeEnum int

const (
	modeList      modeEnum = 0
	modeSearch    modeEnum = 1
	modeEnterPath modeEnum = 2
)

type Model struct {
	// state
	path          string
	previousPath  string
	inputPath     string
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
	name           string
	fileInfo       os.FileInfo
	linkTargetInfo os.FileInfo // != nil when file is a symbolic link
	linkTargetPath string      // != "" when file is a symbolic link
	emphasisTextIx [2]int      // Start and end indexes of emphasis text
	isSelected     bool
	details        ItemDetails
}

type ItemDetails struct {
	Perm, Username, Group, Size, Date string
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
	thiss.previousPath = path
	thiss.width = 80
	thiss.height = 10
	thiss.username = username
	thiss.showDetails = config.SHOW_DETAILS
	thiss.Ls()
	return nil
}

// https://github.com/charmbracelet/bubbletea/blob/master/key.go
var (
	keyEsc           = key.NewBinding(key.WithKeys("esc"))
	keyQuit          = key.NewBinding(key.WithKeys("alt+enter"))
	keyQuitWithoutCd = key.NewBinding(key.WithKeys("ctrl+c"))
	keyUp            = key.NewBinding(key.WithKeys("up"))
	keyDown          = key.NewBinding(key.WithKeys("down"))
	keyLeft          = key.NewBinding(key.WithKeys("left"))
	keyRight         = key.NewBinding(key.WithKeys("right"))
	keyEnter         = key.NewBinding(key.WithKeys("enter"))
	keyTab           = key.NewBinding(key.WithKeys("tab"))
	keySpace         = key.NewBinding(key.WithKeys(" "))
	keyBackspace     = key.NewBinding(key.WithKeys("backspace"))
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
	keySlash         = key.NewBinding(key.WithKeys("/"))
	keyTilde         = key.NewBinding(key.WithKeys("~"))
	keyPrev          = key.NewBinding(key.WithKeys("-"))
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
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println(`cd "` + thiss.path + `"`)
		return thiss, tea.Quit

	case key.Matches(msg, keyEsc) && thiss.mode == modeList:
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

	case key.Matches(msg, keyEnter, keyTab) && (thiss.mode == modeList || thiss.mode == modeSearch):
		shouldExit, hasFailed, exitCmd := thiss.cursorEnter()
		if hasFailed {
			return thiss, nil
		}
		thiss.changeMode(modeList)
		if shouldExit {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Println(exitCmd)
			return thiss, tea.Quit
		}
		return thiss, nil

	// case key.Matches(msg, keySlash) && thiss.mode == modeSearch: // Slash enters directory, on searchMode
	// 	thiss.cursorEnter()
	// 	thiss.changeMode(modeList)
	// 	return thiss, nil

	case key.Matches(msg, keyTilde) && thiss.mode == modeList:
		homePath, _ := os.UserHomeDir()
		thiss.previousPath = thiss.path
		thiss.path = homePath
		thiss.Ls()
		thiss.calculateColsAndRows()
		return thiss, nil

	case key.Matches(msg, keyPrev) && thiss.mode == modeList: // Go to previous path
		var swap = thiss.path
		thiss.path = thiss.previousPath
		thiss.previousPath = swap
		thiss.Ls()
		thiss.calculateColsAndRows()
		thiss.setOffsetToMiddleScreen()
		return thiss, nil

	case key.Matches(msg, keyParent) && thiss.mode == modeList:
		thiss.goParent()
		return thiss, nil

	case key.Matches(msg, keySpace) && (thiss.mode == modeList || thiss.mode == modeSearch):
		// TODO: Selections
		// thiss.toggleSelection()
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

		// Disable modeEnterPath for now
	// case key.Matches(msg, keySlash) && thiss.mode == modeList: // Change to _modeEnterPath
	// 	thiss.changeMode(modeEnterPath)
	// 	return thiss, nil

	case key.Matches(msg, keySlash) && thiss.mode == modeList: // Go to root
		thiss.goToPath("/")
		return thiss, nil

	case msg.Type == tea.KeyRunes && thiss.mode == modeEnterPath:
		thiss.inputPath += string(msg.Runes)
		return thiss, nil

	case key.Matches(msg, keySpace) && thiss.mode == modeEnterPath: // (a Bug? cannot detect space)
		thiss.inputPath += string(" ")
		return thiss, nil

	case key.Matches(msg, keyBackspace) && thiss.mode == modeEnterPath:
		thiss.inputPath = thiss.inputPath[:len(thiss.inputPath)-1]
		if thiss.inputPath == "" {
			thiss.changeMode(modeList)
		}
		return thiss, nil

	case key.Matches(msg, keyEnter) && thiss.mode == modeEnterPath:
		if thiss.isPathOk(thiss.inputPath) {
			thiss.previousPath = thiss.path
			thiss.path = filepath.Clean(thiss.inputPath)
			thiss.Ls()
			thiss.rowOffset = 0
			thiss.cursorIx = 0
			thiss.changeMode(modeList)
		}
		return thiss, nil

	case msg.Type == tea.KeyRunes: // Change to _modeSearch
		if unicode.IsLetter(msg.Runes[0]) && unicode.IsLower(msg.Runes[0]) {
			thiss.searchInput += string(msg.Runes)
			thiss.searchFilter(thiss.searchInput)
			thiss.cursorIx = 0
			thiss.rowOffset = 0
			thiss.changeMode(modeSearch)
		}
		return thiss, nil

	case key.Matches(msg, keyBackspace) && thiss.mode == modeSearch:
		thiss.searchInput = thiss.searchInput[:len(thiss.searchInput)-1]
		thiss.cursorIx = 0
		thiss.rowOffset = 0
		if thiss.searchInput == "" {
			thiss.changeMode(modeList)
			return thiss, nil
		}
		thiss.searchFilter(thiss.searchInput)
		thiss.changeMode(modeSearch)
		return thiss, nil

	case key.Matches(msg, keyEsc, keyClear) && thiss.mode == modeSearch:
		thiss.cursorIx = 0
		thiss.rowOffset = 0
		thiss.changeMode(modeList)
		return thiss, nil
	}
	return thiss, nil
}

func (thiss *Model) View() string {
	if thiss.mode == modeList || thiss.mode == modeSearch {
		return thiss.renderList()
	} else if thiss.mode == modeEnterPath {
		return thiss.renderListScreen(thiss.renderHeader(), "", "")
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
				listOut += "\n" + term.Violet("--More--", false)
				break
			} else if relRow != 0 {
				listOut += "\n"
			}
		}
		if thiss.showDetails {
			listOut += thiss.renderItemWithDetails(item, thiss.items, thiss.cursorIx == ix)
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
	o := thiss.username + ": "
	if thiss.mode == modeList {
		o += thiss.path
	} else if thiss.mode == modeEnterPath {
		if thiss.isPathOk(thiss.inputPath) {
			o += term.Green(thiss.inputPath, false) +
				"\n" +
				term.Gray("Manual path input mode. Please type the desired path.", false) +
				"\n" +
				term.Gray("Press <enter> to enter path", false)
		} else {
			o += term.Red(thiss.inputPath, false) +
				"\n" +
				term.Gray("Invalid path", false) +
				"\n" +
				term.Gray("Fix it or press <esc> to exit path input mode", false)
		}
	} else if thiss.mode == modeSearch {
		o = strings.TrimSuffix(o+thiss.path, "/") + "/" + term.Violet(thiss.searchInput, false)
	}
	return o
}

func renderFooter() string {
	s := "" +
		// "space: Select   shift+c: Copy   alt+x: Cut      alt+v: Paste   del: Delete" +
		// "\n" +
		// "alt+h: Help     esc: Quit       lower: Search   alt+d: Details"
		// "[/] Go to root   [~] Go to home   [alt+backspace] Go to parent" +
		// "\n" +
		"[a-z] Search   [alt+d] Details   [alt+enter] Quit   [ctrl+c] Quit without cd"

	return term.Gray(s, false)
}

func (thiss *Model) Ls() {
	resolvedPath, _ := filepath.EvalSymlinks(thiss.path)
	files, err := os.ReadDir(resolvedPath)
	if err != nil {
		panic(err)
	}
	addRelDirs := func() []Item {
		ret := []Item{}
		oneDot := Item{
			name:     "./",
			fileInfo: nil,
		}
		if config.ADD_ONE_DOT_FOLDER {
			ret = append(ret, oneDot)
		}
		if filepath.Clean(thiss.path) == "/" {
			return ret
		}
		if config.ADD_TWO_DOT_FOLDER {
			previousDirStat, err := os.Stat(filepath.Clean(filepath.Join(thiss.path, "..")))
			if err != nil {
				panic(err)
			}
			var perm, username, group, size, date string = getDetails(previousDirStat)
			ret = append(ret, Item{
				name:     "../",
				fileInfo: previousDirStat,
				details: ItemDetails{
					Perm:     perm,
					Username: username,
					Group:    group,
					Size:     size,
					Date:     date,
				},
			})
		}

		return ret
	}

	thiss.items = addRelDirs()
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			panic(err)
		}
		name := info.Name()
		if info.IsDir() {
			name += "/"
		}
		linkTarget := ""
		var linkTargetInfo os.FileInfo
		if isFileSymlink(info) {
			target, err := os.Readlink(filepath.Join(resolvedPath, name))
			if err != nil {
				panic(err)
			}
			linkTarget = target
			targetResolvedFullPath, _ := filepath.EvalSymlinks(filepath.Join(resolvedPath, name))
			isRelativeTarget := !filepath.IsAbs(targetResolvedFullPath)
			if isRelativeTarget {
				targetResolvedFullPath = filepath.Clean(filepath.Join(resolvedPath, targetResolvedFullPath))
			}
			linkTargetInfo, err = os.Stat(targetResolvedFullPath)
			if err != nil {
				panic(err)
			}
		}
		var perm, username, group, size, date string = getDetails(info)
		thiss.items = append(thiss.items, Item{
			name:     name,
			fileInfo: info,
			details: ItemDetails{
				Perm:     perm,
				Username: username,
				Group:    group,
				Size:     size,
				Date:     date,
			},
			linkTargetInfo: linkTargetInfo,
			linkTargetPath: linkTarget,
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

func isFileExecutable(fileInfo os.FileInfo) bool {
	if fileInfo == nil {
		return false
	}
	return fileInfo.Mode()&0111 != 0
}

func isFileSymlink(fileInfo os.FileInfo) bool {
	return fileInfo.Mode()&os.ModeSymlink != 0
}

func isFileDevice(fileInfo os.FileInfo) bool {
	return fileInfo.Mode()&os.ModeDevice != 0
}

func addColorByFileType(text string, item Item, isFocused bool, marks []int) string {
	// isSymlink := false
	isSymlink := item.linkTargetInfo != nil
	isSymlinkTargetDir := isSymlink && item.linkTargetInfo.IsDir()
	isSymlinkTargetExec := isSymlink && isFileExecutable(item.linkTargetInfo)
	isSymlinkTargetDevice := isSymlink && isFileDevice(item.linkTargetInfo)
	if isFocused {
		text = term.Violet(text, true)
	} else if item.isSelected {
		text = term.Orange(text, true)
	} else if item.fileInfo == nil {
		text = addTextEmphasisAndBlue(text, marks)
	} else if isSymlink {
		text = addTextEmphasisAndCyan(text, marks)
	} else if isFileDevice(item.fileInfo) || (isSymlink && isSymlinkTargetDevice) {
		text = addTextEmphasisAndYellow(text, marks)
	} else if item.fileInfo.IsDir() || (isSymlink && isSymlinkTargetDir) {
		text = addTextEmphasisAndBlue(text, marks)
	} else if isFileExecutable(item.fileInfo) || (isSymlink && isSymlinkTargetExec) {
		text = addTextEmphasisAndGreen(text, marks)
	} else {
		text = addTextEmphasisAndNothing(text, marks)
	}
	return text
}

func addTextEmphasisAndBlue(text string, marks []int) string {
	s1 := text[0:marks[0]]
	s2 := text[marks[0]:marks[1]]
	s3 := text[marks[1]:]
	return term.Blue(s1, false) + term.Emphasis(s2) + term.Blue(s3, false)
}

func addTextEmphasisAndNothing(text string, marks []int) string {
	s1 := text[0:marks[0]]
	s2 := text[marks[0]:marks[1]]
	s3 := text[marks[1]:]
	return s1 + term.Emphasis(s2) + s3
}

func addTextEmphasisAndGreen(text string, marks []int) string {
	s1 := text[0:marks[0]]
	s2 := text[marks[0]:marks[1]]
	s3 := text[marks[1]:]
	return term.Green(s1, false) + term.Emphasis(s2) + term.Green(s3, false)
}

func addTextEmphasisAndYellow(text string, marks []int) string {
	s1 := text[0:marks[0]]
	s2 := text[marks[0]:marks[1]]
	s3 := text[marks[1]:]
	return term.Yellow(s1, false) + term.Emphasis(s2) + term.Yellow(s3, false)
}

func addTextEmphasisAndCyan(text string, marks []int) string {
	s1 := text[0:marks[0]]
	s2 := text[marks[0]:marks[1]]
	s3 := text[marks[1]:]
	return term.Cyan(s1, false) + term.Emphasis(s2) + term.Cyan(s3, false)
}

func getDetails(fileInfo os.FileInfo) (perm, username, group, size, date string) {
	if fileInfo == nil {
		return
	}
	// perm
	perm = fileInfo.Mode().String()
	// User
	var fsys = fileInfo.Sys()
	var uid = fsys.(*syscall.Stat_t).Uid
	var us, _ = user.LookupId(strconv.Itoa(int(uid)))
	username = us.Username
	// Group
	var gid = fsys.(*syscall.Stat_t).Gid
	var gr, _ = user.LookupGroupId(strconv.Itoa(int(gid)))
	group = gr.Name
	// Size
	size = humanize.Bytes(uint64(fileInfo.Size()))
	// date
	date = fileInfo.ModTime().Format("02 Jan 06 15:04") // time.RFC822 minus timezone
	return
}

func getDetailsSizes(all []Item) (permSz, userSz, groupSz, sizeSz, dateSz int) {
	for _, it := range all {
		permSz = max(permSz, len(it.details.Perm))
		userSz = max(userSz, len(it.details.Username))
		groupSz = max(groupSz, len(it.details.Group))
		sizeSz = max(sizeSz, len(it.details.Size))
		dateSz = max(dateSz, len(it.details.Date))
	}
	return
}

func (thiss *Model) renderItem(item Item, isFocused bool) string {
	style := lipgloss.NewStyle().Width(thiss.colSize)
	// Handle symlink colors as the target colors
	isSymlink := item.linkTargetInfo != nil
	if isSymlink {
		item.fileInfo = item.linkTargetInfo
		item.linkTargetInfo = nil
		if item.fileInfo.IsDir() {
			item.name += "/"
		}
	}
	return style.Render(addColorByFileType(item.name, item, isFocused, item.emphasisTextIx[:]))
}

func (thiss *Model) renderItemWithDetails(item Item, all []Item, isFocused bool) string {
	sep := strings.Repeat(" ", config.DETAILS_SEPARATOR_SZ)
	symlinkInfo := ""
	if item.linkTargetInfo != nil {
		itemTarget := item
		itemTarget.fileInfo = item.linkTargetInfo
		itemTarget.linkTargetInfo = nil
		dirSlash := ""
		if itemTarget.fileInfo.IsDir() {
			dirSlash = "/"
		}
		symlinkInfo = " -> " + addColorByFileType(item.linkTargetPath+dirSlash, itemTarget, false, []int{0, 0})
	}
	var permSz, userSz, groupSz, sizeSz, dateSz = getDetailsSizes(all)
	var details = term.Width(item.details.Perm, permSz) + sep +
		term.Width(item.details.Username, userSz) + sep +
		term.Width(item.details.Group, groupSz) + sep +
		term.Width(item.details.Size, sizeSz) + sep +
		term.Width(item.details.Date, dateSz) + sep
	return term.Gray(details, false) + addColorByFileType(item.name, item, isFocused, item.emphasisTextIx[:]) + symlinkInfo
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
	} else if mode == modeEnterPath {
		thiss.inputPath = "/"
		thiss.mode = modeEnterPath
	}
}

func (thiss *Model) searchFilter(input string) {
	filtered := []Item{}
	for _, v := range thiss.dirItems {
		var a = strings.ToLower(input)
		var b = strings.ToLower(v.name)
		foundIx := strings.Index(b, a)
		if foundIx > -1 {
			v.emphasisTextIx[0] = foundIx
			v.emphasisTextIx[1] = foundIx + len(a)
			filtered = append(filtered, v)
		}
		strings.Index(b, a)
	}

	// Rank names that starts with input first
	sort.Slice(filtered, func(i, j int) bool {
		rank := func(it Item) int {
			if strings.HasPrefix(it.name, input) {
				return -1
			}
			return 0
		}
		return rank(filtered[i]) < rank(filtered[j])
	})

	thiss.filteredItems = filtered
}

func (thiss *Model) isPathOk(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
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

func (thiss *Model) cursorEnter() (shouldExit bool, hasFailed bool, exitCmd string) {
	if len(thiss.items) == 0 {
		hasFailed = true
		return
	}
	curItem := thiss.CurrentItem()

	if curItem.name == "./" {
		shouldExit = true
		exitCmd = `cd "` + thiss.path + `"`
		return
	}

	if curItem.name == "../" {
		thiss.goParent()
		return
	}

	if curItem.fileInfo == nil {
		return
	}
	isSymlink := curItem.linkTargetInfo != nil
	fileInfo := curItem.fileInfo
	if isSymlink {
		fileInfo = curItem.linkTargetInfo
	}
	if fileInfo.IsDir() {
		thiss.previousPath = thiss.path
		thiss.path = filepath.Clean(
			filepath.Join(thiss.path, curItem.name),
		)
		thiss.cursorIx = 0
		thiss.rowOffset = 0
		thiss.Ls()
		thiss.calculateColsAndRows()
		return
	}

	if !fileInfo.IsDir() {
		shouldExit = true
		if isFileExecutable(fileInfo) {
			var execCmd = `"./` + curItem.name + `"`
			exitCmd = `cd "` + thiss.path + `" && ` + execCmd
			if config.ADD_LAST_CMD_TO_HISTORY {
				exitCmd += " && history -s " + execCmd
			}
		} else {
			var execCmd = fmt.Sprintf(config.EDIT_FILE_CMD, curItem.name)
			exitCmd = `cd "` + thiss.path + `" && ` + execCmd
			if config.ADD_LAST_CMD_TO_HISTORY {
				exitCmd += " && history -s '" + execCmd + "'"
			}
		}
		return
	}

	return
}

func (thiss *Model) goParent() {
	newPath := filepath.Clean(
		filepath.Join(thiss.path, ".."),
	)
	if newPath == thiss.path {
		return
	}
	prevName := filepath.Base(thiss.path)
	thiss.path = newPath
	thiss.Ls()
	thiss.calculateColsAndRows()
	// Set cursor to the previous open folder
	thiss.cursorIx = func(name string) (cursorFromName int) {
		for k, v := range thiss.items {
			if v.fileInfo != nil && v.fileInfo.Name() == name {
				return k
			}
		}
		return
	}(prevName)
	thiss.setOffsetToMiddleScreen()
}

func (thiss *Model) goToPath(path string) {
	thiss.previousPath = thiss.path
	thiss.path = path
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

// setOffsetToMiddleScreen recalculates and set the offset, to cursor be in the middle of the screen
func (thiss *Model) setOffsetToMiddleScreen() {
	thiss.rowOffset = 0
	thiss.addRowOffset(thiss.cursorRowIx() - (thiss.rowsDisplayed() / 2))
}

func (thiss *Model) toggleDetails() {
	thiss.showDetails = !thiss.showDetails
	thiss.calculateColsAndRows()
	thiss.setOffsetToMiddleScreen()
}
