package viewer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	minColumnWidth    = 6
	maxColumnWidth    = 80
	columnSampleLimit = 200
	headerTableName   = "ERMHDR"
	maxScanTokenSize  = 16 * 1024 * 1024
)

type FileData struct {
	Path   string
	Name   string
	Header Header
	Tables []TableData
}

type Header struct {
	Version     string
	ExportDate  string
	Application string
	Login       string
	User        string
	Database    string
	Module      string
	Currency    string
}

type TableData struct {
	Name           string
	Columns        []string
	RowNumberWidth int
	ColumnWidths   []int
	ContentWidth   int
	rawRows        [][]string
}

type tableBlock struct {
	Name    string
	Columns []string
	Rows    [][]string
}

func LoadFile(path string) (*FileData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return Load(file, path)
}

func Load(r io.Reader, path string) (*FileData, error) {
	header, blocks, err := parseXER(r)
	if err != nil {
		return nil, err
	}

	tables := make([]TableData, 0, len(blocks)+1)
	tables = append(tables, buildHeaderTable(header))
	for _, block := range blocks {
		table := TableData{
			Name:    block.Name,
			Columns: append([]string(nil), block.Columns...),
			rawRows: block.Rows,
		}
		table.finalizeLayout()
		tables = append(tables, table)
	}

	return &FileData{
		Path:   path,
		Name:   filepath.Base(path),
		Header: header,
		Tables: tables,
	}, nil
}

func parseXER(r io.Reader) (Header, []tableBlock, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)

	var (
		header       Header
		blocks       []tableBlock
		current      *tableBlock
		blockOrdinal = map[string]int{}
	)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, headerTableName+"\t") || line == headerTableName:
			header = parseHeaderLine(line)
		case strings.HasPrefix(line, "%T"):
			name := ""
			parts := splitTabs(line)
			if len(parts) > 1 {
				name = parts[1]
			}
			if name == "" {
				current = nil
				continue
			}
			current = &tableBlock{Name: name}
			blocks = append(blocks, *current)
			current = &blocks[len(blocks)-1]
		case strings.HasPrefix(line, "%F"):
			if current == nil {
				continue
			}
			fields := splitTabs(line)
			if len(fields) > 1 {
				current.Columns = append([]string(nil), fields[1:]...)
			} else {
				current.Columns = nil
			}
		case strings.HasPrefix(line, "%R"):
			if current == nil {
				continue
			}
			values := splitTabs(line)
			if len(values) > 1 {
				values = values[1:]
			} else {
				values = nil
			}

			if len(current.Columns) == 0 {
				current.Columns = synthesizeColumns(len(values))
			}
			values = padRow(values, len(current.Columns))
			current.Rows = append(current.Rows, values)
		}
	}

	if err := scanner.Err(); err != nil {
		return Header{}, nil, fmt.Errorf("read XER: %w", err)
	}

	for i := range blocks {
		name := blocks[i].Name
		blockOrdinal[name]++
		if blockOrdinal[name] > 1 {
			blocks[i].Name = fmt.Sprintf("%s (%d)", name, blockOrdinal[name])
		}
		if len(blocks[i].Columns) == 0 {
			blocks[i].Columns = synthesizeColumns(maxRowWidth(blocks[i].Rows))
		}
		for rowIndex := range blocks[i].Rows {
			blocks[i].Rows[rowIndex] = padRow(blocks[i].Rows[rowIndex], len(blocks[i].Columns))
		}
	}

	return header, blocks, nil
}

func parseHeaderLine(line string) Header {
	parts := splitTabs(line)
	header := Header{}
	if len(parts) > 1 {
		header.Version = parts[1]
	}
	if len(parts) > 2 {
		header.ExportDate = formatHeaderDate(parts[2])
	}
	if len(parts) > 3 {
		header.Application = parts[3]
	}
	if len(parts) > 4 {
		header.Login = parts[4]
	}
	if len(parts) > 5 {
		header.User = parts[5]
	}
	if len(parts) > 6 {
		header.Database = parts[6]
	}
	if len(parts) > 7 {
		header.Module = parts[7]
	}
	if len(parts) > 8 {
		header.Currency = parts[8]
	}
	return header
}

func buildHeaderTable(header Header) TableData {
	rows := [][]string{
		{"version", header.Version},
		{"export_date", header.ExportDate},
		{"application", header.Application},
		{"login", header.Login},
		{"user", header.User},
		{"database", header.Database},
		{"module", header.Module},
		{"currency", header.Currency},
	}
	table := TableData{
		Name:    headerTableName,
		Columns: []string{"field", "value"},
		rawRows: rows,
	}
	table.finalizeLayout()
	return table
}

func (t TableData) RowCount() int {
	return len(t.rawRows)
}

func (t TableData) ColumnCount() int {
	return len(t.Columns)
}

func (t TableData) Cell(row, col int) string {
	if row < 0 || row >= len(t.rawRows) || col < 0 || col >= len(t.Columns) {
		return ""
	}
	if col >= len(t.rawRows[row]) {
		return ""
	}
	return t.rawRows[row][col]
}

func (t TableData) maxHorizontalOffset(visibleWidth int) int {
	if visibleWidth <= 0 || t.ContentWidth <= visibleWidth {
		return 0
	}
	return t.ContentWidth - visibleWidth
}

func (t *TableData) finalizeLayout() {
	t.RowNumberWidth = max(3, len(strconv.Itoa(max(1, t.RowCount()))))
	t.ColumnWidths = make([]int, len(t.Columns))

	sampleRows := min(t.RowCount(), columnSampleLimit)
	for col, name := range t.Columns {
		width := max(minColumnWidth, runeLen(name))
		for row := 0; row < sampleRows; row++ {
			width = max(width, runeLen(t.Cell(row, col)))
		}
		t.ColumnWidths[col] = min(width, maxColumnWidth)
	}

	if len(t.Columns) == 0 {
		t.ContentWidth = 0
		return
	}

	width := 0
	for i, colWidth := range t.ColumnWidths {
		width += colWidth
		if i > 0 {
			width += 3
		}
	}
	t.ContentWidth = width
}

func splitTabs(line string) []string {
	return strings.Split(line, "\t")
}

func padRow(values []string, width int) []string {
	if len(values) >= width {
		return values
	}
	row := make([]string, width)
	copy(row, values)
	return row
}

func synthesizeColumns(width int) []string {
	columns := make([]string, width)
	for i := range columns {
		columns[i] = fmt.Sprintf("col_%d", i+1)
	}
	return columns
}

func maxRowWidth(rows [][]string) int {
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	return width
}

func formatHeaderDate(raw string) string {
	if raw == "" || strings.EqualFold(raw, "n") {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed.Format("2006-01-02")
	}
	return raw
}
