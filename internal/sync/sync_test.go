package sync

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	rep := Sync(Options{Source: src, Target: dst})
	if rep.Copied != 1 || rep.Overwritten != 0 || len(rep.Errors) != 0 {
		t.Fatalf("unexpected rep after first sync: %+v", *rep)
	}

	time.Sleep(1100 * time.Millisecond)
	mustWrite(t, filepath.Join(src, "a.txt"), "hello world")

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

func writeWithModTime(t *testing.T, path string, data string, perm os.FileMode, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), perm); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Set modification time (atime = now, mtime = provided mtime)
	if err := os.Chtimes(path, time.Now(), mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestCopyFile_NewFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	mtime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	content := "hello world"
	writeWithModTime(t, src, content, 0o640, mtime)

	info, err := os.Stat(src)
	if err != nil {
		t.Fatalf("stat src: %v", err)
	}

	if err := copyFile(src, dst, info); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(b) != content {
		t.Fatalf("content mismatch: %q", string(b))
	}

	dstInfo, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}

	// Check size
	if dstInfo.Size() != int64(len(content)) {
		t.Fatalf("size mismatch: %d", dstInfo.Size())
	}

	// Permissions (may differ on Windows)
	if runtime.GOOS != "windows" {
		if dstInfo.Mode().Perm() != info.Mode().Perm() {
			t.Fatalf("perm mismatch: got %v want %v", dstInfo.Mode().Perm(), info.Mode().Perm())
		}
	}

	// Modification time (second resolution)
	if !dstInfo.ModTime().Truncate(time.Second).Equal(mtime) {
		t.Fatalf("mtime mismatch: got %v want %v", dstInfo.ModTime(), mtime)
	}
}

func TestCopyFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	writeWithModTime(t, src, "NEW DATA", 0o600, time.Now().Add(-1*time.Hour).Truncate(time.Second))
	if err := os.WriteFile(dst, []byte("OLD"), 0o600); err != nil {
		t.Fatalf("prepare dst: %v", err)
	}

	info, _ := os.Stat(src)
	if err := copyFile(src, dst, info); err != nil {
		t.Fatalf("copyFile overwrite: %v", err)
	}

	b, _ := os.ReadFile(dst)
	if string(b) != "NEW DATA" {
		t.Fatalf("overwrite failed: %q", string(b))
	}
}

func TestCopyFile_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "no-such.txt")
	dst := filepath.Join(dir, "dst.txt")

	dummyInfoFile := filepath.Join(dir, "dummy.txt")
	if err := os.WriteFile(dummyInfoFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("dummy write: %v", err)
	}
	info, _ := os.Stat(dummyInfoFile)

	err := copyFile(src, dst, info)
	if err == nil || !strings.Contains(err.Error(), "open src") {
		t.Fatalf("expected open src error, got: %v", err)
	}
}

func TestCopyFile_RenameFailureCleansTemp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst") // Create a directory with that name so rename fails

	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatalf("mkdir dst (dir): %v", err)
	}

	writeWithModTime(t, src, "data", 0o600, time.Now().Add(-30*time.Minute).Truncate(time.Second))
	info, _ := os.Stat(src)

	err := copyFile(src, dst, info)
	if err == nil || !strings.Contains(err.Error(), "rename") {
		t.Fatalf("expected rename error, got: %v", err)
	}

	tmpPath := dst + ".tmp~"
	if _, statErr := os.Stat(tmpPath); !os.IsNotExist(statErr) {
		t.Fatalf("temp file not cleaned up: err=%v", statErr)
	}
}
