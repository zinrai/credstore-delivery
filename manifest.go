package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// parseManifest reads what to deliver, a list of secret names, and nothing
// about where: the hosts come with -hosts, from whatever already lists them.
func parseManifest(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	var names []string
	if err := yaml.Unmarshal(data, &names); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("manifest %s lists no secret names", path)
	}
	for _, name := range names {
		if !secretNamePattern.MatchString(name) {
			return nil, fmt.Errorf("%q is not a secret name: segments of letters, digits, '_' and '-' joined by '/'", name)
		}
	}
	return names, nil
}

// parseHosts does not check the names' form: putRemote passes them to ssh
// after "--", so none is read as an option.
func parseHosts(flag string) ([]string, error) {
	hosts := strings.Split(flag, ",")
	if slices.Contains(hosts, "") {
		return nil, fmt.Errorf("-hosts %q has an empty host", flag)
	}
	return hosts, nil
}
