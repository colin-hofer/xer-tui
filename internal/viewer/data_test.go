package viewer

import (
	"strings"
	"testing"
)

func TestLoadParsesRawXERBlocksWithoutExternalSchema(t *testing.T) {
	t.Helper()

	const sample = "" +
		"ERMHDR\t19.12\t2026-04-02\txv-test\talice\tAlice\tSandbox\tProject Management\tUSD\n" +
		"%T\tPROJECT\n" +
		"%F\tproj_id\tproj_short_name\tlast_recalc_date\n" +
		"%R\t42\tDEMO\t2026-04-01 09:30\n" +
		"%T\tTASK\n" +
		"%F\ttask_id\tproj_id\ttask_name\tstatus_code\n" +
		"%R\t7\t42\tExcavate\tTK_NotStart\n" +
		"%R\t8\t42\tPour Concrete\tTK_Active\n"

	data, err := Load(strings.NewReader(sample), "sample.xer")
	if err != nil {
		t.Fatalf("load sample XER: %v", err)
	}

	if got, want := data.Header.Version, "19.12"; got != want {
		t.Fatalf("header version = %q, want %q", got, want)
	}
	if got, want := data.Header.ExportDate, "2026-04-02"; got != want {
		t.Fatalf("header export date = %q, want %q", got, want)
	}

	projectTable := findTable(data.Tables, "PROJECT")
	if projectTable == nil {
		t.Fatalf("project table not found")
	}
	if got, want := projectTable.RowCount(), 1; got != want {
		t.Fatalf("project rows = %d, want %d", got, want)
	}
	if got, want := projectTable.Cell(0, findColumn(*projectTable, "proj_short_name")), "DEMO"; got != want {
		t.Fatalf("project short name = %q, want %q", got, want)
	}

	taskTable := findTable(data.Tables, "TASK")
	if taskTable == nil {
		t.Fatalf("task table not found")
	}
	if got, want := taskTable.RowCount(), 2; got != want {
		t.Fatalf("task rows = %d, want %d", got, want)
	}
	if got, want := taskTable.Cell(1, findColumn(*taskTable, "task_name")), "Pour Concrete"; got != want {
		t.Fatalf("task name = %q, want %q", got, want)
	}

	headerTable := findTable(data.Tables, headerTableName)
	if headerTable == nil {
		t.Fatalf("header table not found")
	}
	if got, want := headerTable.Cell(2, 1), "xv-test"; got != want {
		t.Fatalf("header application = %q, want %q", got, want)
	}
}

func TestLoadHandlesRowsBeforeFields(t *testing.T) {
	t.Helper()

	const sample = "" +
		"%T\tTASK\n" +
		"%R\t7\t42\tExcavate\n"

	data, err := Load(strings.NewReader(sample), "odd.xer")
	if err != nil {
		t.Fatalf("load odd XER: %v", err)
	}

	taskTable := findTable(data.Tables, "TASK")
	if taskTable == nil {
		t.Fatalf("task table not found")
	}
	if got, want := taskTable.Columns, []string{"col_1", "col_2", "col_3"}; len(got) != len(want) {
		t.Fatalf("task columns len = %d, want %d", len(got), len(want))
	}
	if got, want := taskTable.Cell(0, 2), "Excavate"; got != want {
		t.Fatalf("task col_3 = %q, want %q", got, want)
	}
}

func findTable(tables []TableData, name string) *TableData {
	for i := range tables {
		if tables[i].Name == name {
			return &tables[i]
		}
	}
	return nil
}

func findColumn(table TableData, name string) int {
	for i, column := range table.Columns {
		if column == name {
			return i
		}
	}
	return -1
}
