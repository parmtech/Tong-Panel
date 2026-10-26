//go:build linux

// Package agent is a non-privileged local-socket protocol experiment.
// It has NO host-mutating verbs. Never run this proof of concept as root.
package agent

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
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
	req := Request{}
	d := json.NewDecoder(io.LimitReader(c, 512))
	d.DisallowUnknownFields()
	if err = d.Decode(&req); err != nil {
		return err
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
	if _, err := os.Lstat(path); err == nil {
		return nil, errors.New("refusing to replace existing socket")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	addr := &net.UnixAddr{Name: path, Net: "unix"}
	old := syscall.Umask(0077)
	defer syscall.Umask(old)
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
