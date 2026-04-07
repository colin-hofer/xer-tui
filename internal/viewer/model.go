package viewer

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"xer-tui/internal/update"
	"xer-tui/internal/version"
)

type keyMap struct {
	NextTable   key.Binding
	PrevTable   key.Binding
	Down        key.Binding
	Up          key.Binding
	PageDown    key.Binding
	PageUp      key.Binding
	Right       key.Binding
	Left        key.Binding
	FastRight   key.Binding
	FastLeft    key.Binding
	Top         key.Binding
	Bottom      key.Binding
	Home        key.Binding
	Search      key.Binding
	FilterTable key.Binding
	NextMatch   key.Binding
	PrevMatch   key.Binding
	Update      key.Binding
	Help        key.Binding
	Quit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextTable, k.Down, k.Right, k.Search, k.FilterTable, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextTable, k.PrevTable, k.Down, k.Up, k.PageDown, k.PageUp},
		{k.Right, k.Left, k.FastRight, k.FastLeft, k.Home, k.Top, k.Bottom},
		{k.Search, k.FilterTable, k.NextMatch, k.PrevMatch, k.Update, k.Help, k.Quit},
	}
}

type Model struct {
	data *FileData

	width  int
	height int

	tableIndex   int
	tableScroll  int
	selectedRow  int
	rowScroll    int
	columnScroll int

	searchMode      string // "", "row", "table"
	searchInput     textinput.Model
	filteredIndices []int
	rowQuery        string
	matchedRows     []int

	checkingUpdate  bool
	UpdateRequested bool
	status          string

	showHelp bool
	keys     keyMap
	help     help.Model
}

type styles struct {
	Title         lipgloss.Style
	Muted         lipgloss.Style
	SidebarTitle  lipgloss.Style
	SidebarItem   lipgloss.Style
	SidebarActive lipgloss.Style
	Status        lipgloss.Style
	Header        lipgloss.Style
	SelectedRow   lipgloss.Style
	Empty         lipgloss.Style
}

var appStyles = styles{
	Title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
	Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	SidebarTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	SidebarItem:   lipgloss.NewStyle().Padding(0, 1),
	SidebarActive: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")),
	Status:        lipgloss.NewStyle().Bold(true),
	Header:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
	SelectedRow:   lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")),
	Empty:         lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
}

func NewModel(data *FileData) Model {
	h := help.New()
	h.ShowAll = false

	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 64

	return Model{
		data:        data,
		searchInput: si,
		keys: keyMap{
			NextTable: key.NewBinding(key.WithKeys("tab", "]"), key.WithHelp("tab", "next table")),
			PrevTable: key.NewBinding(key.WithKeys("shift+tab", "["), key.WithHelp("shift+tab", "prev table")),
			Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "next row")),
			Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "prev row")),
			PageDown:  key.NewBinding(key.WithKeys("pgdown", "d"), key.WithHelp("pgdn", "page down")),
			PageUp:    key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
			Right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/right", "scroll right")),
			Left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/left", "scroll left")),
			FastRight: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "jump right")),
			FastLeft:  key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "jump left")),
			Home:      key.NewBinding(key.WithKeys("0"), key.WithHelp("0", "left edge")),
			Top:       key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
			Bottom:    key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
			Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search rows")),
			FilterTable: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "filter tables")),
			NextMatch:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
			PrevMatch:   key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
			Update:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update xv")),
			Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
		help: h,
	}
}

type updateCheckMsg struct {
	Result update.Result
	Err    error
}

func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		updater, err := update.New(update.Config{
			RepoOwner:      version.RepositoryOwner,
			RepoName:       version.RepositoryName,
			BinaryName:     version.BinaryName,
			CurrentVersion: version.Current(),
		})
		if err != nil {
			return updateCheckMsg{Err: err}
		}
		result, err := updater.Check(context.Background())
		return updateCheckMsg{Result: result, Err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.clamp()
		return m, nil

	case updateCheckMsg:
		m.checkingUpdate = false
		if typed.Err != nil {
			m.status = typed.Err.Error()
			return m, nil
		}
		if !typed.Result.Available {
			m.status = fmt.Sprintf("already up to date (%s)", displayVersion(typed.Result.LatestVersion))
			return m, nil
		}
		m.status = fmt.Sprintf("update %s -> %s available, closing to install...",
			displayVersion(typed.Result.PreviousVersion),
			displayVersion(typed.Result.LatestVersion))
		m.UpdateRequested = true
		return m, tea.Quit

	case tea.KeyMsg:
		if m.status != "" {
			m.status = ""
		}

		if m.searchMode != "" {
			return m.updateSearch(typed)
		}

		switch {
		case key.Matches(typed, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(typed, m.keys.Search):
			m.searchMode = "row"
			m.searchInput.Prompt = "/"
			m.searchInput.SetValue("")
			return m, m.searchInput.Focus()
		case key.Matches(typed, m.keys.FilterTable):
			m.searchMode = "table"
			m.searchInput.Prompt = "t:"
			m.searchInput.SetValue("")
			m.filteredIndices = nil
			m.tableScroll = 0
			return m, m.searchInput.Focus()
		case key.Matches(typed, m.keys.NextMatch):
			m.jumpToMatch(1)
		case key.Matches(typed, m.keys.PrevMatch):
			m.jumpToMatch(-1)
		case key.Matches(typed, m.keys.Update):
			if m.checkingUpdate {
				return m, nil
			}
			m.status = "checking latest release..."
			m.checkingUpdate = true
			return m, checkForUpdateCmd()
		case key.Matches(typed, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
		case key.Matches(typed, m.keys.NextTable):
			m.setTable(m.visibleTablePos() + 1)
		case key.Matches(typed, m.keys.PrevTable):
			m.setTable(m.visibleTablePos() - 1)
		case key.Matches(typed, m.keys.Down):
			m.moveRow(1)
		case key.Matches(typed, m.keys.Up):
			m.moveRow(-1)
		case key.Matches(typed, m.keys.PageDown):
			m.moveRow(max(1, m.rowsVisible()))
		case key.Matches(typed, m.keys.PageUp):
			m.moveRow(-max(1, m.rowsVisible()))
		case key.Matches(typed, m.keys.Right):
			m.moveColumn(4)
		case key.Matches(typed, m.keys.Left):
			m.moveColumn(-4)
		case key.Matches(typed, m.keys.FastRight):
			m.moveColumn(max(10, m.tableViewportWidth()/2))
		case key.Matches(typed, m.keys.FastLeft):
			m.moveColumn(-max(10, m.tableViewportWidth()/2))
		case key.Matches(typed, m.keys.Home):
			m.columnScroll = 0
		case key.Matches(typed, m.keys.Top):
			m.selectedRow = 0
			m.rowScroll = 0
		case key.Matches(typed, m.keys.Bottom):
			if rows := m.currentTable().RowCount(); rows > 0 {
				m.selectedRow = rows - 1
				m.ensureRowVisible()
			}
		}

		m.clamp()
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchInput.Blur()
		if m.searchMode == "table" {
			if len(m.filteredIndices) > 0 {
				m.tableIndex = m.filteredIndices[0]
			}
			m.selectedRow = 0
			m.rowScroll = 0
			m.columnScroll = 0
		} else if m.searchMode == "row" {
			m.rowQuery = m.searchInput.Value()
			m.buildRowMatches()
			m.jumpToMatch(0)
		}
		m.searchMode = ""
		m.clamp()
		return m, nil
	case "esc":
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		if m.searchMode == "table" {
			m.filteredIndices = nil
			m.tableScroll = 0
		}
		m.searchMode = ""
		m.clamp()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchMode == "table" {
		m.updateTableFilter()
	}
	return m, cmd
}

func (m *Model) updateTableFilter() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredIndices = nil
		return
	}
	m.filteredIndices = make([]int, 0)
	for i, table := range m.data.Tables {
		if strings.Contains(strings.ToLower(table.Name), query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
	if len(m.filteredIndices) > 0 {
		m.tableIndex = m.filteredIndices[0]
		m.selectedRow = 0
		m.rowScroll = 0
		m.columnScroll = 0
	}
	m.tableScroll = 0
}

func (m *Model) buildRowMatches() {
	m.matchedRows = m.matchedRows[:0:0]
	query := strings.ToLower(m.rowQuery)
	if query == "" {
		return
	}
	table := m.currentTable()
	for row := 0; row < table.RowCount(); row++ {
		for col := 0; col < table.ColumnCount(); col++ {
			if strings.Contains(strings.ToLower(table.Cell(row, col)), query) {
				m.matchedRows = append(m.matchedRows, row)
				break
			}
		}
	}
}

func (m *Model) jumpToMatch(direction int) {
	if len(m.matchedRows) == 0 {
		m.status = "no matches"
		return
	}

	if direction == 0 {
		m.selectedRow = m.matchedRows[0]
		m.ensureRowVisible()
		m.status = fmt.Sprintf("match 1/%d", len(m.matchedRows))
		return
	}

	target := -1
	if direction > 0 {
		for _, r := range m.matchedRows {
			if r > m.selectedRow {
				target = r
				break
			}
		}
		if target == -1 {
			target = m.matchedRows[0]
		}
	} else {
		for i := len(m.matchedRows) - 1; i >= 0; i-- {
			if m.matchedRows[i] < m.selectedRow {
				target = m.matchedRows[i]
				break
			}
		}
		if target == -1 {
			target = m.matchedRows[len(m.matchedRows)-1]
		}
	}

	m.selectedRow = target
	m.ensureRowVisible()

	pos := 0
	for i, r := range m.matchedRows {
		if r == target {
			pos = i + 1
			break
		}
	}
	m.status = fmt.Sprintf("match %d/%d", pos, len(m.matchedRows))
}

func (m Model) visibleTables() []int {
	if m.filteredIndices != nil {
		return m.filteredIndices
	}
	all := make([]int, len(m.data.Tables))
	for i := range all {
		all[i] = i
	}
	return all
}

func (m Model) visibleTablePos() int {
	for i, idx := range m.visibleTables() {
		if idx == m.tableIndex {
			return i
		}
	}
	return 0
}

func displayVersion(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.width < 60 || m.height < 10 {
		return appStyles.Empty.Render("window too small for xv")
	}

	headerLine := appStyles.Title.Render(fmt.Sprintf("xv  %s", m.data.Name))
	if m.status != "" {
		headerLine += "  " + appStyles.Muted.Render(m.status)
	}
	contentHeight := m.height - 2
	if contentHeight < 4 {
		contentHeight = 4
	}

	sidebarWidth := m.sidebarWidth()
	mainWidth := max(20, m.width-sidebarWidth-1)

	left := m.renderSidebar(contentHeight, sidebarWidth)
	right := m.renderMain(contentHeight, mainWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var footer string
	if m.searchMode != "" {
		footer = m.searchInput.View()
	} else {
		footer = m.help.View(m.keys)
	}
	return lipgloss.JoinVertical(lipgloss.Left, headerLine, body, footer)
}

func (m *Model) currentTable() TableData {
	if len(m.data.Tables) == 0 {
		return TableData{}
	}
	return m.data.Tables[m.tableIndex]
}

func (m *Model) setTable(index int) {
	tables := m.visibleTables()
	if len(tables) == 0 {
		return
	}

	switch {
	case index < 0:
		index = len(tables) - 1
	case index >= len(tables):
		index = 0
	}

	m.tableIndex = tables[index]
	m.selectedRow = 0
	m.rowScroll = 0
	m.columnScroll = 0
	if m.rowQuery != "" {
		m.buildRowMatches()
	}
	m.ensureTableVisible()
}

func (m *Model) moveRow(delta int) {
	rows := m.currentTable().RowCount()
	if rows == 0 {
		m.selectedRow = 0
		m.rowScroll = 0
		return
	}

	m.selectedRow += delta
	if m.selectedRow < 0 {
		m.selectedRow = 0
	}
	if m.selectedRow >= rows {
		m.selectedRow = rows - 1
	}
	m.ensureRowVisible()
}

func (m *Model) moveColumn(delta int) {
	m.columnScroll += delta
}

func (m *Model) clamp() {
	if len(m.data.Tables) == 0 {
		return
	}

	if m.tableIndex < 0 {
		m.tableIndex = 0
	}
	if m.tableIndex >= len(m.data.Tables) {
		m.tableIndex = len(m.data.Tables) - 1
	}

	table := m.currentTable()
	rows := table.RowCount()
	if rows == 0 {
		m.selectedRow = 0
		m.rowScroll = 0
	} else {
		if m.selectedRow < 0 {
			m.selectedRow = 0
		}
		if m.selectedRow >= rows {
			m.selectedRow = rows - 1
		}
		maxRowScroll := max(0, rows-m.rowsVisible())
		if m.rowScroll < 0 {
			m.rowScroll = 0
		}
		if m.rowScroll > maxRowScroll {
			m.rowScroll = maxRowScroll
		}
	}

	maxColumnScroll := table.maxHorizontalOffset(m.tableViewportWidth())
	if m.columnScroll < 0 {
		m.columnScroll = 0
	}
	if m.columnScroll > maxColumnScroll {
		m.columnScroll = maxColumnScroll
	}

	m.ensureRowVisible()
	m.ensureTableVisible()
}

func (m *Model) ensureRowVisible() {
	visible := m.rowsVisible()
	if visible <= 0 {
		m.rowScroll = 0
		return
	}
	if m.selectedRow < m.rowScroll {
		m.rowScroll = m.selectedRow
	}
	if m.selectedRow >= m.rowScroll+visible {
		m.rowScroll = m.selectedRow - visible + 1
	}
}

func (m *Model) ensureTableVisible() {
	pos := m.visibleTablePos()
	visible := max(1, m.contentHeight()-1)
	if pos < m.tableScroll {
		m.tableScroll = pos
	}
	if pos >= m.tableScroll+visible {
		m.tableScroll = pos - visible + 1
	}
}

func (m Model) sidebarWidth() int {
	maxWidth := 24
	for _, table := range m.data.Tables {
		label := m.tableLabel(table)
		maxWidth = max(maxWidth, runeLen(label)+2)
	}
	return min(maxWidth, 34)
}

func (m Model) contentHeight() int {
	return max(4, m.height-2)
}

func (m Model) rowsVisible() int {
	return max(1, m.contentHeight()-3)
}

func (m Model) tableViewportWidth() int {
	sidebarWidth := m.sidebarWidth()
	mainWidth := max(20, m.width-sidebarWidth-1)
	table := m.currentTable()
	fixedWidth := table.RowNumberWidth + 3
	return max(1, mainWidth-fixedWidth)
}

func (m Model) renderSidebar(height, width int) string {
	lines := make([]string, 0, height)
	tables := m.visibleTables()

	title := "Tables"
	if len(m.filteredIndices) > 0 {
		title = fmt.Sprintf("Tables (%d)", len(m.filteredIndices))
	}
	lines = append(lines, fitToWidth(appStyles.SidebarTitle.Render(title), width))

	visible := max(1, height-1)
	end := min(len(tables), m.tableScroll+visible)
	for si := m.tableScroll; si < end; si++ {
		realIndex := tables[si]
		label := fitToWidth(m.tableLabel(m.data.Tables[realIndex]), width)
		style := appStyles.SidebarItem
		if realIndex == m.tableIndex {
			style = appStyles.SidebarActive
		}
		lines = append(lines, style.Width(width).Render(label))
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderMain(height, width int) string {
	table := m.currentTable()
	lines := make([]string, 0, height)

	status := fmt.Sprintf(
		"%s  rows:%d  cols:%d  row:%d/%d  x:%d",
		table.Name,
		table.RowCount(),
		table.ColumnCount(),
		min(table.RowCount(), m.selectedRow+1),
		max(1, table.RowCount()),
		m.columnScroll,
	)
	lines = append(lines, fitToWidth(appStyles.Status.Render(status), width))

	headerLine, separator := m.renderHeader(table, width)
	lines = append(lines, headerLine, separator)

	rowLimit := m.rowsVisible()
	for row := 0; row < rowLimit; row++ {
		rowIndex := m.rowScroll + row
		if rowIndex >= table.RowCount() {
			lines = append(lines, strings.Repeat(" ", width))
			continue
		}

		line := m.renderRow(table, rowIndex, width)
		if rowIndex == m.selectedRow {
			line = appStyles.SelectedRow.Width(width).Render(line)
		}
		lines = append(lines, line)
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHeader(table TableData, width int) (string, string) {
	fixedWidth := table.RowNumberWidth + 3
	dataWidth := max(1, width-fixedWidth)

	prefix := fmt.Sprintf("%*s | ", table.RowNumberWidth, "#")
	data := make([]string, len(table.Columns))
	for i, name := range table.Columns {
		data[i] = padOrTrim(name, table.ColumnWidths[i])
	}
	dataText := strings.Join(data, " | ")

	header := prefix + clipPad(dataText, m.columnScroll, dataWidth)
	separator := strings.Repeat("-", fixedWidth) + strings.Repeat("-", min(dataWidth, max(0, runeLen(dataText)-m.columnScroll)))

	return appStyles.Header.Width(width).Render(fitToWidth(header, width)), fitToWidth(separator, width)
}

func (m Model) renderRow(table TableData, row int, width int) string {
	fixedWidth := table.RowNumberWidth + 3
	dataWidth := max(1, width-fixedWidth)

	prefix := fmt.Sprintf("%*d | ", table.RowNumberWidth, row+1)
	data := make([]string, len(table.Columns))
	for col := range table.Columns {
		data[col] = padOrTrim(table.Cell(row, col), table.ColumnWidths[col])
	}
	dataText := strings.Join(data, " | ")
	return fitToWidth(prefix+clipPad(dataText, m.columnScroll, dataWidth), width)
}

func (m Model) tableLabel(table TableData) string {
	return fmt.Sprintf("%-14s %6d", table.Name, table.RowCount())
}

func runeLen(s string) int {
	return len([]rune(s))
}

func clipPad(s string, start, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if start >= len(runes) {
		return strings.Repeat(" ", width)
	}

	end := min(len(runes), start+width)
	out := string(runes[start:end])
	if pad := width - runeLen(out); pad > 0 {
		out += strings.Repeat(" ", pad)
	}
	return out
}

func fitToWidth(s string, width int) string {
	return clipPad(s, 0, width)
}

func padOrTrim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width == 1 {
			return string(runes[:1])
		}
		return string(runes[:width-1]) + "…"
	}
	if pad := width - len(runes); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

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

