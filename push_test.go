package main

import "testing"

func TestOnlyTheExpectedPutLineIsAccepted(t *testing.T) {
	if got, err := parseResult("[update] web.token\n", "web.token"); err != nil || got != resultUpdate {
		t.Fatalf("got %q %v", got, err)
	}
	for _, out := range []string{"", "[update] other\n", "[deleted] web.token\n", "[update] web.token\nextra\n"} {
		if _, err := parseResult(out, "web.token"); err == nil {
			t.Errorf("accepted %q", out)
		}
	}
}
