package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// credstoreDir is fixed, not a flag: it is the directory systemd itself
// uses for encrypted credentials, and the drop-ins name the same path.
const credstoreDir = "/etc/credstore.encrypted"

const (
	resultCreate    = "create"
	resultUpdate    = "update"
	resultUnchanged = "unchanged"
)

func runPut(args []string) error {
	flags := flag.NewFlagSet("put", flag.ExitOnError)
	name := flags.String("name", "", "Credential name (required); the value is read from stdin")
	flags.Parse(args)
	if *name == "" {
		flags.Usage()
		return fmt.Errorf("-name is required")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("put must be run as root: the host key is readable by root only")
	}
	value, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(value) == 0 {
		return fmt.Errorf("no value on stdin")
	}

	result, err := putCredential(systemdCreds{}, credstoreDir, *name, value)
	if err != nil {
		return err
	}
	fmt.Printf("[%s] %s\n", result, *name)
	return nil
}

type cipher interface {
	encrypt(name string, plain []byte) ([]byte, error)
	decrypt(name, path string) ([]byte, error)
}

// putCredential decrypts and compares, not compares ciphertexts: encryption
// is randomized, so the same value never encrypts to the same bytes, and a
// rewrite would trigger the path unit on every run.
// putCredential checks the name itself, not leaving it to the caller: it
// writes as root, and an account allowed to sudo only put must not reach
// past dir with a name such as "../x".
func putCredential(c cipher, dir, name string, value []byte) (string, error) {
	if !credentialNamePattern.MatchString(name) {
		return "", fmt.Errorf("%q is not a credential name: segments of letters, digits, '_' and '-' joined by '.'", name)
	}
	path := filepath.Join(dir, name)
	result := resultCreate
	if _, err := os.Lstat(path); err == nil {
		current, err := c.decrypt(name, path)
		if err != nil {
			return "", fmt.Errorf("%s exists but does not decrypt; remove it by hand if the host key changed: %w", path, err)
		}
		if bytes.Equal(current, value) {
			return resultUnchanged, nil
		}
		result = resultUpdate
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	encrypted, err := c.encrypt(name, value)
	if err != nil {
		return "", err
	}
	return result, replaceFile(dir, name, encrypted)
}

// replaceFile writes a temporary file and renames it over the target, not
// writing in place: a path unit watching the target would otherwise fire on
// a half-written file, and a crash would leave one behind.
func replaceFile(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}

type systemdCreds struct{}

// encrypt writes to stdout, not to a file: the file mode and the rename
// stay in replaceFile.
func (systemdCreds) encrypt(name string, plain []byte) ([]byte, error) {
	return runCreds(plain, "encrypt", "--name="+name, "--with-key=host", "-", "-")
}

func (systemdCreds) decrypt(name, path string) ([]byte, error) {
	return runCreds(nil, "decrypt", "--name="+name, path, "-")
}

func runCreds(stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("systemd-creds", args...)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemd-creds %s: %w", args[0], err)
	}
	return out, nil
}
