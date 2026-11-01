package svcpaths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAtomicWriteTarget(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "directory")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "missing target is allowed", path: filepath.Join(dir, "new")},
		{name: "regular file is allowed", path: regular},
		{name: "symlink is rejected", path: link, wantErr: true},
		{name: "directory is rejected", path: subdir, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAtomicWriteTarget(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAtomicWriteTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
