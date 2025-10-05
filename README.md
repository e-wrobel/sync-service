# filesync – one-shot folder synchronization (source → target)

CLI tool written in Go. On startup it compares *source* and *target* folders and:

- copies files missing in target,
- overwrites files that differ by **size** or **modification time**,
- optionally deletes files present only in target (`--delete-missing`).

Errors are logged, but do **not** stop the run.

## Requirements
- Go 1.18+
- Works on Linux/macOS/Windows (uses only Go stdlib)

## Build
```bash
g o build -o sync-service ./cmd/sync
```

## Usage
```bash
  ./sync-service --source /path/to/src --target /path/to/dst [--delete-missing]
```

### Examples
Copy/overwrite only:
```bash
  ./sync-service --source ./example/src --target ./example/dst
```

Also remove files missing in source:
```bash
  ./sync-service --source ./example/src --target ./example/dst --delete-missing
```

### Exit codes
- `0` – completed without errors
- `1` – completed with non-fatal errors (they were logged)
- `2` – invalid CLI usage (missing args etc.)

## Notes & design
- Comparison uses size or mod-time (rounded to seconds for cross-FS stability).
- Only regular files are synchronized. Non-regular entries are logged and skipped.
- Overwrites are **atomic**: data is written to a temporary file and then `os.Rename` replaces the target.

## Tests
```bash
  go test ./...
```

## Makefile overview

| Target  | Description                                         |
|---------|-----------------------------------------------------|
| build   | Compile the binary                                  |
| run     | Run the binary with optional CLI arguments         |
| test    | Run all tests                                       |
| cover   | Run tests with coverage report                      |
| fmt     | Format Go source files                              |
| imports | Fix import statements                               |
| lint    | Run linters to check code quality                   |
| mod     | Tidy and verify Go modules                          |
| graph   | Generate dependency graph                           |

### Examples

Build the binary:
```bash
make build
```

Run the binary with CLI arguments:
```bash
make run ARGS="--source ./example/src --target ./example/dst --delete-missing"
```

## Quick demo (with included example folders)

After building the binary, try syncing the included example folders:

```bash
# Run sync without deleting extra files
  ./sync-service --source ./example/src --target ./example/dst

# Run sync and also delete files missing in source
  ./sync-service --source ./example/src --target ./example/dst --delete-missing
```

Initial state:
- `example/src/` contains `hello.txt`, `data.csv`
- `example/dst/` contains `old.txt`

After first run (without --delete-missing):
- `hello.txt` and `data.csv` copied into `dst/`
- `old.txt` remains

After run with --delete-missing:
- `old.txt` is removed from `dst/`
