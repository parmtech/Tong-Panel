// Package preflight does read-only compatibility checks, not an installer.
package preflight

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type Report struct {
	OS, Version, Arch string
	Candidate         bool
	Warnings          []string
}

func ParseOSRelease(raw string) (string, string, error) {
	fields := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		a, b, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[a] = strings.Trim(b, "\"'")
	}
	if err := sc.Err(); err != nil {
		return "", "", err
	}
	id, v := fields["ID"], fields["VERSION_ID"]
	if id == "" || v == "" {
		return "", "", errors.New("missing ID/VERSION_ID")
	}
	return id, v, nil
}
func Detect(path string) (Report, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Report{}, err
	}
	id, v, err := ParseOSRelease(string(b))
	if err != nil {
		return Report{}, err
	}
	report := Report{OS: id, Version: v, Arch: runtime.GOARCH}
	report.Candidate = runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && ((id == "ubuntu" && v == "24.04") || (id == "debian" && v == "12"))
	report.Warnings = append(report.Warnings, "Development-only preflight: no distribution has passed full installation certification")
	if !report.Candidate {
		report.Warnings = append(report.Warnings, fmt.Sprintf("OS %s %s / %s is not a tested candidate", id, v, runtime.GOARCH))
	}
	return report, nil
}
