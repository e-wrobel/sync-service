package validators

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustDir(t *testing.T) {
	type test struct {
		name    string
		prepare func(t *testing.T) string
		wantErr bool
	}

	tests := []test{
		{
			name: "ok_directory",
			prepare: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "not_directory_file",
			prepare: func(t *testing.T) string {
				dir := t.TempDir()
				fpath := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(fpath, []byte("data"), 0o600); err != nil {
					t.Fatalf("write file: %v", err)
				}
				return fpath
			},
			wantErr: true,
		},
		{
			name: "missing_path",
			prepare: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does_not_exist")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.prepare(t)
			err := MustDir(path)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
