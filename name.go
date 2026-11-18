package main

import (
	"regexp"
	"strings"
)

var (
	// secretNamePattern is the contract with the socket. No '.': '/' becomes
	// '.' in the credential name, and with '.' allowed "a/b" and "a.b" would
	// land in the same file.
	secretNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$`)
	// credentialNamePattern is the image of secretNamePattern; it also keeps
	// '/', "." and ".." out of the path under credstoreDir.
	credentialNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)
)

// credentialName maps a secret name to the name systemd sees: '/' is not
// allowed in a credential name, which is also a file name.
func credentialName(secret string) string {
	return strings.ReplaceAll(secret, "/", ".")
}
