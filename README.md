# xer-tui

`xer-tui` is a terminal UI for browsing Primavera XER files as tables.

It opens an `.xer` file, splits it into tables, and lets you scroll through rows and columns in a dataframe-style view.

## Install

From this repo:

```bash
go install ./cmd/xv
```

This installs the binary as:

```bash
/Users/mrityunjay/go/bin/xv
```

## Run

```bash
xv /path/to/file.xer
```

Example:

```bash
xv "/Users/mrityunjay/dev/work/accurise/syncify/tests/data/xer/Northwell Health - Zucker - UPD-10 - Export.xer"
```

You can also run it without installing:

```bash
go run ./cmd/xv /path/to/file.xer
```

## Controls

- `tab` / `shift+tab`: switch tables
- `j` / `k`: move up and down
- `h` / `l`: scroll left and right
- `pgup` / `pgdn`: page up and down
- `g` / `G`: jump to top or bottom
- `?`: toggle help
- `q`: quit

## Notes

- The parser is local to this repo and does not import Syncify.
- The viewer reads raw XER blocks directly from `ERMHDR`, `%T`, `%F`, and `%R` lines.
- Unknown tables are still shown as long as they exist in the XER file.
