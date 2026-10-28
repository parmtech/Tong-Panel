// tongctl doctor is a read-only developer diagnostic, NOT a safe installation command.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/parmtech/tong-panel/internal/preflight"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bootstrap-owner" {
		if err := localOwnerBootstrap(os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "local bootstrap:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "preflight" {
		snapshot, err := preflight.DetectSnapshot("/etc/os-release")
		if err != nil {
			fmt.Fprintln(os.Stderr, "preflight:", err)
			os.Exit(1)
		}
		allowed, err := runInstallPreflight(os.Args[2:], snapshot, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "preflight:", err)
			os.Exit(2)
		}
		if !allowed {
			os.Exit(3)
		}
		return
	}
	if len(os.Args) != 2 || os.Args[1] != "doctor" {
		fmt.Fprintln(os.Stderr, "usage: tongctl doctor | tongctl preflight --min-disk-bytes BYTES --required-tcp-ports PORT[,PORT...] | tongctl bootstrap-owner --db PATH --username NAME --tenant NAME --password-stdin")
		os.Exit(2)
	}
	r, err := preflight.Detect("/etc/os-release")
	if err != nil {
		fmt.Fprintln(os.Stderr, "preflight:", err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
}

type installPreflightResult struct {
	Notice string                 `json:"notice"`
	Host   preflight.Report       `json:"host"`
	Gate   preflight.MutationGate `json:"gate"`
}

// runInstallPreflight evaluates caller-supplied thresholds against a read-only
// snapshot. An allowed result is only a facts/policy match; it never authorizes
// or performs installation or any other host mutation.
func runInstallPreflight(args []string, snapshot preflight.Snapshot, output io.Writer) (bool, error) {
	flags := flag.NewFlagSet("preflight", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	minimumDisk := flags.Uint64("min-disk-bytes", 0, "operator-selected minimum free disk bytes")
	ports := flags.String("required-tcp-ports", "", "comma-separated TCP ports that must be unused")
	if err := flags.Parse(args); err != nil || len(flags.Args()) != 0 || *minimumDisk == 0 || strings.TrimSpace(*ports) == "" {
		return false, errors.New("usage: tongctl preflight --min-disk-bytes BYTES --required-tcp-ports PORT[,PORT...]")
	}
	var requiredPorts []uint16
	seen := make(map[uint16]bool)
	for _, raw := range strings.Split(*ports, ",") {
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 16)
		if err != nil || value == 0 || seen[uint16(value)] {
			return false, errors.New("required TCP ports must be unique integers from 1 through 65535")
		}
		seen[uint16(value)] = true
		requiredPorts = append(requiredPorts, uint16(value))
	}
	report, err := preflight.Evaluate(snapshot)
	if err != nil {
		report = preflight.Report{Warnings: []string{"Host facts could not be evaluated"}}
	}
	gate := preflight.EvaluateMutationGate(snapshot, preflight.MutationRequirements{
		MinimumDiskBytes: *minimumDisk, RequiredTCPPorts: requiredPorts,
	})
	result := installPreflightResult{
		Notice: "Read-only development gate. Allowed means only that detected facts match supplied thresholds; it is not installation authorization or support certification.",
		Host: report, Gate: gate,
	}
	if err := json.NewEncoder(output).Encode(result); err != nil {
		return false, err
	}
	return gate.Allowed, nil
}
