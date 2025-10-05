# Sync-Service – one-shot folder synchronization (source → target)

CLI tool written in Go. On startup it compares *source* and *target* folders and:

- copies files missing in target,
- overwrites files that differ by **size** or **modification time**,
- optionally deletes files present only in target (`--delete-missing`).

Errors are logged, but do **not** stop the run.

## Requirements
- Go 1.18+
- Works on Linux/macOS/Windows (uses only Go stdlib)

## Manual build
```bash
  go build -o sync-service ./cmd/sync
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

## Data integrity and atomic operations
The synchronization process uses safe write operations to ensure data integrity. 
Files are first copied to a temporary file and only then atomically renamed to the target location. 
This approach prevents corruption from partial writes during the copy process. 
Additionally, modification times are preserved using `os.Chtimes`, which allows for accurate comparison of file timestamps 
during subsequent runs. This method avoids race conditions and inconsistent states if the program crashes mid-copy. 
As a result, this design makes the tool robust and reliable for cron-based or automated runs.

### Data flow diagram (Mermaid)

```mermaid
flowchart LR
    A["Source file"] --> B["Copy to temp file (.tmp~)"]
    B --> C["Preserve mod-time (os.Chtimes)"]
    C --> D["Atomic rename temp to destination"]
    D --> E["Target file updated safely"]

```
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

## Quick demo (with included example folders)
Remark: First we need to build the binary.

After building the binary, try syncing the included example folders:

### Run sync without deleting extra files
```bash
  ./sync-service --source ./example/src --target ./example/dst
```

### Run sync and also delete files missing in source
```bash
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

## Scheduling with cron (user mode)

To run sync-service automatically every 10 minutes using cron, follow these steps:

1. Build the binary (if not already built):
```bash
  go build -o sync-service ./cmd/sync
```

2. Edit your crontab file by running:
```bash
  crontab -e
```

3. Add the following line to schedule the sync every 10 minutes:
```
*/10 * * * * /usr/local/bin/sync-service --source /home/marcin/data/src --target /home/marcin/data/dst >> /home/marcin/sync-service/sync.log 2>&1
```

Explanation of the cron line:
- `*/10 * * * *` — runs the command every 10 minutes.
- `/usr/local/bin/sync-service` — the full path to the sync-service binary.
- `--source /home/marcin/data/src` — specifies the source directory.
- `--target /home/marcin/data/dst` — specifies the target directory.
- `>> /home/marcin/sync-service/sync.log 2>&1` — appends both standard output and error output to the `sync.log` file for logging.

If you want the sync to also delete files missing in the source directory, add the `--delete-missing` flag to the command:
```
*/10 * * * * /user/local/bin/sync-service --source /home/marcin/data/src --target /home/marcin/data/dst --delete-missing >> /home/marcin/sync-service/sync.log 2>&1
```

## Debugging and troubleshooting
You can run application in debug mode in Goland.
Configuration is provided in `.run` directory.