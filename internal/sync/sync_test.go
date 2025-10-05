package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustWrite(t *testing.T, path string, data string) os.FileInfo {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	return fi
}

func TestCopyNewAndOverwrite(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	s1info := mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	rep := Sync(Options{Source: src, Target: dst})
	if rep.Copied != 1 || rep.Overwritten != 0 || len(rep.Errors) != 0 {
		t.Fatalf("unexpected rep after first sync: %+v", *rep)
	}

	time.Sleep(1100 * time.Millisecond)
	_ = s1info
	_ = mustWrite(t, filepath.Join(src, "a.txt"), "hello world")

	rep2 := Sync(Options{Source: src, Target: dst})
	if rep2.Overwritten != 1 {
		t.Fatalf("expected overwrite=1, got %+v", *rep2)
	}
}

func TestDeleteMissing(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	mustWrite(t, filepath.Join(src, "x.txt"), "x")
	mustWrite(t, filepath.Join(dst, "only-in-dst.txt"), "y")

	rep := Sync(Options{Source: src, Target: dst, DeleteMissing: false})
	if rep.Copied != 1 || rep.Deleted != 0 {
		t.Fatalf("unexpected rep: %+v", *rep)
	}
	if _, err := os.Stat(filepath.Join(dst, "only-in-dst.txt")); err != nil {
		t.Fatalf("expected file to remain, err=%v", err)
	}

	rep2 := Sync(Options{Source: src, Target: dst, DeleteMissing: true})
	if rep2.Deleted != 1 {
		t.Fatalf("expected deleted=1, got %+v", *rep2)
	}
	if _, err := os.Stat(filepath.Join(dst, "only-in-dst.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, err=%v", err)
	}
}
