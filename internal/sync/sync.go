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

func Sync(opt Options) *Report {
	if opt.Logger == nil {
		opt.Logger = log.Default()
	}
	rep := &Report{}

	err := filepath.WalkDir(opt.Source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			opt.Logger.Printf("ERR: odczyt %s: %v", path, err)
			rep.addErr(err)
			return nil
		}
		if path == opt.Source {
			return nil
		}
		rel, _ := filepath.Rel(opt.Source, path)
		targetPath := filepath.Join(opt.Target, rel)

		if d.IsDir() {
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
			opt.Logger.Printf("SKIP: nieregularny plik %s (mode=%v)", path, info.Mode())
			rep.Skipped++
			return nil
		}

		tst, err := os.Stat(targetPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
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
			if err := copyFile(path, targetPath, info); err != nil {
				opt.Logger.Printf("ERR: overwrite %s -> %s: %v", path, targetPath, err)
				rep.addErr(err)
				return nil
			}
			opt.Logger.Printf("OVERWRITE: %s -> %s", path, targetPath)
			rep.Overwritten++
		} else {
			opt.Logger.Printf("SKIP: %s (identical)", rel)
			rep.Skipped++
		}
		return nil
	})
	if err != nil {
		opt.Logger.Printf("ERR: walk %s: %v", opt.Source, err)
		rep.addErr(err)
	}

	if opt.DeleteMissing {
		err = filepath.WalkDir(opt.Target, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				opt.Logger.Printf("ERR: odczyt %s: %v", path, err)
				rep.addErr(err)
				return nil
			}
			if path == opt.Target {
				return nil
			}
			rel, _ := filepath.Rel(opt.Target, path)
			srcPath := filepath.Join(opt.Source, rel)

			if d.IsDir() {
				return nil
			}

			if _, err := os.Stat(srcPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
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

	return rep
}

func differ(src, dst os.FileInfo) bool {
	if src.Size() != dst.Size() {
		return true
	}
	return !truncateToSeconds(src.ModTime()).Equal(truncateToSeconds(dst.ModTime()))
}

func truncateToSeconds(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

func copyFile(srcPath, dstPath string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dstPath), err)
	}

	sf, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer sf.Close()

	tmp := dstPath + ".tmp~"
	df, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}

	_, cErr := io.Copy(df, sf)
	cCloseErr := df.Close()
	if cErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("copy: %w", cErr)
	}
	if cCloseErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", cCloseErr)
	}

	if err := os.Chtimes(tmp, time.Now(), srcInfo.ModTime()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("chtimes: %w", err)
	}

	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
