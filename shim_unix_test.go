//go:build unix

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistShimPATHSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := t.TempDir()
	src := filepath.Join(bin, "dsh")
	if err := os.WriteFile(src, []byte("#!/bin/sh\n# "+shimMarker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := persistShimPATHImpl(bin); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(home, ".local", "bin", "dsh")
	got, err := os.Readlink(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("link = %s, want %s", got, src)
	}
}

func TestPersistShimPATHDoesNotClobber(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := t.TempDir()
	src := filepath.Join(bin, "dsh")
	if err := os.WriteFile(src, []byte("#!/bin/sh\n# "+shimMarker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	destDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "dsh")
	if err := os.WriteFile(dest, []byte("#!/bin/sh\nreal dsh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := persistShimPATHImpl(bin); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "#!/bin/sh\nreal dsh\n" {
		t.Fatalf("clobbered: %s", b)
	}
}
