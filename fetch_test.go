package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// socketServer serves values on a Unix socket
func socketServer(t *testing.T, values map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "s.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, ok := values[r.URL.Path[1:]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(value))
	}))
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return path
}

func TestValuesAreFetchedByteForByte(t *testing.T) {
	values, err := fetchAll(socketClient(socketServer(t, map[string]string{"web/token": "v\n"})), []string{"web/token"})
	if err != nil {
		t.Fatal(err)
	}
	if string(values["web/token"]) != "v\n" {
		t.Fatalf("got %q", values["web/token"])
	}
}

func TestAnyMissingNameStopsTheRun(t *testing.T) {
	client := socketClient(socketServer(t, map[string]string{"web/token": "v"}))
	if _, err := fetchAll(client, []string{"web/token", "web/missing"}); err == nil {
		t.Fatal("accepted")
	}
}
