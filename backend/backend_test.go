package backend

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEmbeddedBinaryReadable(t *testing.T) {
	data, err := PythonBackend.ReadFile(embedFilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", embedFilePath, err)
	}
	if len(data) == 0 {
		t.Fatal("embedded binary is empty")
	}
}

func TestEmbeddedHashDeterministic(t *testing.T) {
	data, err := PythonBackend.ReadFile(embedFilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	h1 := fmt.Sprintf("%x", sha256.Sum256(data))
	h2 := fmt.Sprintf("%x", sha256.Sum256(data))
	if h1 != h2 {
		t.Fatalf("hash mismatch: %s vs %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("unexpected hash length %d", len(h1))
	}
}

func TestGetExecutablePath_ExtractsAndReuses(t *testing.T) {
	tmpDir := t.TempDir()
	orig := userCacheDir
	userCacheDir = func() (string, error) { return tmpDir, nil }
	defer func() { userCacheDir = orig }()

	path1, err := GetExecutablePath(PythonBackend)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	info, err := os.Stat(path1)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("extracted binary is empty")
	}

	data, _ := PythonBackend.ReadFile(embedFilePath)
	ondisk, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path1, err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(ondisk)) != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("on-disk hash does not match embedded hash")
	}

	path2, err := GetExecutablePath(PythonBackend)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if path1 != path2 {
		t.Fatalf("cache miss: %q != %q", path1, path2)
	}
}

func TestGetExecutablePath_ReplacesStaleBinary(t *testing.T) {
	tmpDir := t.TempDir()
	orig := userCacheDir
	userCacheDir = func() (string, error) { return tmpDir, nil }
	defer func() { userCacheDir = orig }()

	origPath, err := GetExecutablePath(PythonBackend)
	if err != nil {
		t.Fatalf("initial extract: %v", err)
	}

	if err := os.WriteFile(origPath, []byte("corrupted"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	repairedPath, err := GetExecutablePath(PythonBackend)
	if err != nil {
		t.Fatalf("repair call: %v", err)
	}
	if repairedPath != origPath {
		t.Fatalf("path changed after repair: %q != %q", repairedPath, origPath)
	}

	data, _ := PythonBackend.ReadFile(embedFilePath)
	ondisk, _ := os.ReadFile(repairedPath)
	if fmt.Sprintf("%x", sha256.Sum256(ondisk)) != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("repaired binary hash does not match embedded hash")
	}
}

func TestWriteBinaryAtomically(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "testbin")
	payload := []byte("#!/bin/sh\necho hello\n")
	hash := fmt.Sprintf("%x", sha256.Sum256(payload))

	if err := writeBinaryAtomically(payload, dir, target, hash); err != nil {
		t.Fatalf("writeBinaryAtomically: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("content mismatch")
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(target)
		if info.Mode().Perm()&0111 == 0 {
			t.Fatal("binary is not executable")
		}
	}
}
