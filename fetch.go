package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

// No timeout: the backend's own failure, with its reason, must reach the operator.
func socketClient(socket string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socket)
			},
		},
	}
}

// fetchAll takes every value before any host is touched: a name the socket
// cannot serve stops the run instead of leaving some hosts updated.
func fetchAll(client *http.Client, names []string) (map[string][]byte, error) {
	values := make(map[string][]byte, len(names))
	for _, name := range names {
		value, err := fetch(client, name)
		if err != nil {
			return nil, err
		}
		values[name] = value
	}
	return values, nil
}

func fetch(client *http.Client, name string) ([]byte, error) {
	res, err := client.Get("http://socket/" + name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return io.ReadAll(res.Body)
	case http.StatusNotFound:
		return nil, fmt.Errorf("%s: not found in the secret store", name)
	default:
		return nil, fmt.Errorf("%s: socket answered %s", name, res.Status)
	}
}
