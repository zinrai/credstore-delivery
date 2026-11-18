package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runPush(args []string) error {
	flags := flag.NewFlagSet("push", flag.ExitOnError)
	socket := flags.String("socket", "", "Unix socket serving the values (required)")
	manifestPath := flags.String("manifest", "", "YAML list of the secret names to deliver (required)")
	hostsFlag := flags.String("hosts", "", "Comma-separated hosts to deliver to (required)")
	user := flags.String("user", "", "Account to log in to the hosts with, allowed to sudo credstore-delivery put (required)")
	flags.Parse(args)
	if *socket == "" || *manifestPath == "" || *hostsFlag == "" || *user == "" {
		flags.Usage()
		return fmt.Errorf("-socket, -manifest, -hosts and -user are required")
	}

	names, err := parseManifest(*manifestPath)
	if err != nil {
		return err
	}
	hosts, err := parseHosts(*hostsFlag)
	if err != nil {
		return err
	}
	values, err := fetchAll(socketClient(*socket), names)
	if err != nil {
		return err
	}
	return pushAll(hosts, *user, names, values)
}

// pushAll goes on past a failed host and reports every one at the end, not
// stopping at the first: the other hosts would be left on the old value.
func pushAll(hosts []string, user string, names []string, values map[string][]byte) error {
	var failed []string
	for _, host := range hosts {
		if !pushHost(host, user, names, values) {
			failed = append(failed, host)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed hosts: %s", strings.Join(failed, ","))
	}
	return nil
}

func pushHost(host, user string, names []string, values map[string][]byte) bool {
	ok := true
	for _, name := range names {
		result, err := putRemote(host, user, credentialName(name), values[name])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", host, name, err)
			ok = false
			continue
		}
		fmt.Printf("%s [%s] %s\n", host, result, name)
	}
	return ok
}

// putRemote passes the value on stdin, never as an argument: arguments show
// up in the process list and in the remote shell's history.
// sudo -n, not sudo: a password prompt would hang with stdin taken.
// -l, not User in an ssh configuration: the account holds only the
// privilege put needs, so it is not the one other tools log in with.
func putRemote(host, user, credential string, value []byte) (string, error) {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-l", user, "--", host,
		"sudo", "-n", "credstore-delivery", "put", "-name", credential)
	cmd.Stdin = bytes.NewReader(value)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseResult(string(out), credential)
}

func parseResult(out, credential string) (string, error) {
	for _, result := range []string{resultCreate, resultUpdate, resultUnchanged} {
		if out == "["+result+"] "+credential+"\n" {
			return result, nil
		}
	}
	return "", fmt.Errorf("unexpected output from credstore-delivery put: %q", out)
}
