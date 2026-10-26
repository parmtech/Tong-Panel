//go:build linux

// Package agent is a non-privileged local-socket protocol experiment.
// It has NO host-mutating verbs. Never run this proof of concept as root.
package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Request struct {
	Op string `json:"op"`
}
type Reply struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ServeConn performs a per-connection Linux kernel peer UID check before decoding.
// uid is the explicitly allowed local UID, not a user-provided request field.
func ServeConn(c *net.UnixConn, uid uint32) error {
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}
	raw, err := c.SyscallConn()
	if err != nil {
		return err
	}
	var cred *syscall.Ucred
	var sockerr error
	if err = raw.Control(func(fd uintptr) {
		cred, sockerr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return err
	}
	if sockerr != nil {
		return sockerr
	}
	if cred == nil || cred.Uid != uid {
		return errors.New("peer UID not allowed")
	}
	// Exactly one newline-delimited JSON frame, capped at 512 bytes.
	// Requiring framing avoids waiting for EOF while clients await a reply.
	frame, err := bufio.NewReaderSize(c, 513).ReadBytes('\n')
	if err != nil {
		return err
	}
	if len(frame) > 512 {
		return errors.New("agent frame too large")
	}
	var req Request
	d := json.NewDecoder(bytes.NewReader(frame))
	d.DisallowUnknownFields()
	if err = d.Decode(&req); err != nil {
		return err
	}
	if d.More() {
		return errors.New("multiple agent messages")
	}
	if req.Op != "ping" {
		return errors.New("operation not allowed")
	}
	return json.NewEncoder(c).Encode(Reply{OK: true, Message: "pong"})
}

// Listen creates socket with restrictive permissions. Parent directory must be private.
func Listen(path string) (*net.UnixListener, error) {
	if path == "" {
		return nil, errors.New("empty socket path")
	}
	// Never change the process-global umask: other goroutines may concurrently
	// create sensitive files. Require a non-symlink, private, owned directory.
	parent := filepath.Dir(path)
	st, err := os.Lstat(parent)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("socket parent must be private directory")
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); !ok || sys.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("socket parent must belong to the running user")
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("refusing to replace existing socket")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	addr := &net.UnixAddr{Name: path, Net: "unix"}
	ln, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		ln.Close()
		return nil, err
	}
	return ln, nil
}
