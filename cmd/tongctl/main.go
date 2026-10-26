// tongctl doctor is a read-only developer diagnostic, NOT a safe installation command.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/parmtech/tong-panel/internal/preflight"
	"os"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "doctor" {
		fmt.Fprintln(os.Stderr, "usage: tongctl doctor (read only)")
		os.Exit(2)
	}
	r, err := preflight.Detect("/etc/os-release")
	if err != nil {
		fmt.Fprintln(os.Stderr, "preflight:", err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
}
