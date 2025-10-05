package sync

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	Source        string
	Target        string
	DeleteMissing bool
	Logger        *log.Logger
}

// Sync performs a one-way synchronization from the source directory to the target directory.
// It copies new and modified files from source to target and optionally deletes files in the target
// that are missing from the source.
func Sync(opt Options) *Report {
	// Initialize logger if not provided
	if opt.Logger == nil {
		opt.Logger = log.Default()
	}
	rep := &Report{}

	// Walk through the source directory tree
	err := filepath.WalkDir(opt.Source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			opt.Logger.Printf("ERR: read %s: %v", path, err)
			rep.addErr(err)
			return nil
		}
		if path == opt.Source {
			return nil
		}
		rel, _ := filepath.Rel(opt.Source, path)
		targetPath := filepath.Join(opt.Target, rel)

		if d.IsDir() {
			// Create directories in target as needed
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				opt.Logger.Printf("ERR: mkdir %s: %v", targetPath, err)
				rep.addErr(err)
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			opt.Logger.Printf("ERR: info %s: %v", path, err)
			rep.addErr(err)
			return nil
		}
		if !info.Mode().IsRegular() {
			opt.Logger.Printf("SKIP: not regular file %s (mode=%v)", path, info.Mode())
			rep.Skipped++
			return nil
		}

		tst, err := os.Stat(targetPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// Copy new files that do not exist in target
				if err := copyFile(path, targetPath, info); err != nil {
					opt.Logger.Printf("ERR: copy NEW %s -> %s: %v", path, targetPath, err)
					rep.addErr(err)
					return nil
				}
				opt.Logger.Printf("COPY: %s -> %s", path, targetPath)
				rep.Copied++
				return nil
			}
			opt.Logger.Printf("ERR: stat %s: %v", targetPath, err)
			rep.addErr(err)
			return nil
		}

		if differ(info, tst) {
			// Overwrite files that differ between source and target
			if err := copyFile(path, targetPath, info); err != nil {
				opt.Logger.Printf("ERR: overwrite %s -> %s: %v", path, targetPath, err)
				rep.addErr(err)
				return nil
			}
			opt.Logger.Printf("OVERWRITE: %s -> %s", path, targetPath)
			rep.Overwritten++
		} else {
			// Skip files that are identical
			opt.Logger.Printf("SKIP: %s (identical)", rel)
			rep.Skipped++
		}
		return nil
	})
	if err != nil {
		opt.Logger.Printf("ERR: walk %s: %v", opt.Source, err)
		rep.addErr(err)
	}

	// If DeleteMissing flag is set, remove files in target that are missing from source
	if opt.DeleteMissing {
		err = filepath.WalkDir(opt.Target, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				opt.Logger.Printf("ERR: read %s: %v", path, err)
				rep.addErr(err)
				return nil
			}
			if path == opt.Target {
				return nil
			}
			rel, _ := filepath.Rel(opt.Target, path)
			srcPath := filepath.Join(opt.Source, rel)

			if d.IsDir() {
				// Skip directories during delete pass
				return nil
			}

			// Check if corresponding source file exists
			if _, err := os.Stat(srcPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					// Remove file from target if missing in source
					if rmErr := os.Remove(path); rmErr != nil {
						opt.Logger.Printf("ERR: delete %s: %v", path, rmErr)
						rep.addErr(rmErr)
						return nil
					}
					opt.Logger.Printf("DELETE: %s (missing in source)", path)
					rep.Deleted++
					return nil
				}
				opt.Logger.Printf("ERR: stat %s: %v", srcPath, err)
				rep.addErr(err)
			}
			return nil
		})
		if err != nil {
			opt.Logger.Printf("ERR: walk target %s: %v", opt.Target, err)
			rep.addErr(err)
		}
	}

	// Return report summarizing the synchronization process
	return rep
}

// differ reports whether two files should be treated as different for synchronization.
// It first compares sizes; if sizes are equal, it compares modification times truncated to seconds.
// Truncation avoids false positives due to differing filesystem timestamp precision (e.g., FAT, some network mounts).
// Returns true if files differ by size or (rounded) mod-time.
func differ(src, dst os.FileInfo) bool {
	// Fast path: any size mismatch means we must copy/overwrite.
	if src.Size() != dst.Size() {
		return true
	}
	// Sizes equal: compare modification times, rounded to whole seconds for cross-FS stability.
	return !truncateToSeconds(src.ModTime()).Equal(truncateToSeconds(dst.ModTime()))
}

// truncateToSeconds returns time truncated to whole seconds.
// Some filesystems (e.g., FAT, certain network mounts) store mtimes with second-level
// precision only. Truncation prevents spurious overwrites when comparing timestamps
// coming from filesystems with different time resolutions.
func truncateToSeconds(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

func copyFile(srcPath, dstPath string, srcInfo os.FileInfo) error {
	// Ensure destination directory exists (idempotent).
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
	}

	// Open source file for reading.
	sf, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer sf.Close()

	// Write into a temporary file next to the destination to enable atomic replace.
	tmp := dstPath + ".tmp~"
	df, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}

	// Stream copy data from source to temp; avoid loading whole file into memory.
	_, cErr := io.Copy(df, sf)
	// Close temp file before further metadata operations and rename.
	cCloseErr := df.Close()
	if cErr != nil {
		// Best-effort cleanup of leftover temp file on error.
		_ = os.Remove(tmp)
		return fmt.Errorf("copy: %w", cErr)
	}
	if cCloseErr != nil {
		// Best-effort cleanup of leftover temp file on error.
		_ = os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", cCloseErr)
	}

	// Preserve source modification time on the newly written file (helps future differ()).
	if err := os.Chtimes(tmp, time.Now(), srcInfo.ModTime()); err != nil {
		// Best-effort cleanup of leftover temp file on error.
		_ = os.Remove(tmp)
		return fmt.Errorf("chtimes: %w", err)
	}

	// Atomically replace (or create) destination by renaming temp -> dst.
	if err := os.Rename(tmp, dstPath); err != nil {
		// Best-effort cleanup of leftover temp file on error.
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	// Success: temp replaced destination; nothing else to do.
	return nil
}
