package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeCipher stands in for systemd-creds. Like it, it never encrypts the
// same value to the same bytes, and binds the ciphertext to the name.
type fakeCipher struct{ calls int }

func (c *fakeCipher) encrypt(name string, plain []byte) ([]byte, error) {
	c.calls++
	return append([]byte(name+"\x00"+string(rune('0'+c.calls))+"\x00"), plain...), nil
}

func (c *fakeCipher) decrypt(name, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	prefix := name + "\x00"
	if len(data) < len(prefix)+2 || string(data[:len(prefix)]) != prefix {
		return nil, errors.New("bad credential")
	}
	return data[len(prefix)+2:], nil
}

func put(t *testing.T, c cipher, dir, value string) string {
	t.Helper()
	result, err := putCredential(c, dir, "web.token", []byte(value))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSameValueLeavesTheFileUntouched(t *testing.T) {
	dir, c := t.TempDir(), &fakeCipher{}
	put(t, c, dir, "v1")
	before, _ := os.ReadFile(filepath.Join(dir, "web.token"))
	if got := put(t, c, dir, "v1"); got != resultUnchanged {
		t.Fatalf("got %s, want unchanged", got)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "web.token"))
	if string(before) != string(after) {
		t.Fatal("file rewritten")
	}
}

func TestResultTellsCreateFromUpdate(t *testing.T) {
	dir, c := t.TempDir(), &fakeCipher{}
	if got := put(t, c, dir, "v1"); got != resultCreate {
		t.Fatalf("got %s, want create", got)
	}
	if got := put(t, c, dir, "v2"); got != resultUpdate {
		t.Fatalf("got %s, want update", got)
	}
	if got, _ := c.decrypt("web.token", filepath.Join(dir, "web.token")); string(got) != "v2" {
		t.Fatalf("stored %q, want v2", got)
	}
}

func TestUndecryptableFileIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "web.token")
	os.WriteFile(path, []byte("garbage"), 0o600)
	if _, err := putCredential(&fakeCipher{}, dir, "web.token", []byte("v")); err == nil {
		t.Fatal("accepted")
	}
	if data, _ := os.ReadFile(path); string(data) != "garbage" {
		t.Fatal("file overwritten")
	}
}

func TestNameCannotReachPastTheDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "credstore.encrypted")
	for _, name := range []string{"../escaped", "web/token", ".", "..", ""} {
		if _, err := putCredential(&fakeCipher{}, dir, name, []byte("v")); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Fatalf("wrote %v", entries)
	}
}
