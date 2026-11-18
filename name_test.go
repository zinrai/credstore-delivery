package main

import "testing"

// The drop-ins name the credential as the README says, so the mapping is
// the contract, not only a name that put accepts.
func TestSlashBecomesDotInTheCredentialName(t *testing.T) {
	c := credentialName("web/db-password")
	if c != "web.db-password" {
		t.Fatalf("web/db-password maps to %s, want web.db-password", c)
	}
	if !credentialNamePattern.MatchString(c) {
		t.Fatalf("put rejects %s", c)
	}
}
