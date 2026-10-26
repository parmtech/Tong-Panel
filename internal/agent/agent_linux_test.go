//go:build linux

package agent

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestOnlyPingAndPeerUID(t *testing.T) {
	for _, tc := range []struct {
		payload string
		uid     uint32
		want    bool
	}{{`{"op":"ping"}`, uint32(os.Getuid()), true}, {`{"op":"root.exec"}`, uint32(os.Getuid()), false}, {`{"op":"ping","cmd":"whoami"}`, uint32(os.Getuid()), false}, {`{"op":"ping"}`, uint32(os.Getuid() + 1000), false}} {
		path := filepath.Join(t.TempDir(), "agent.sock")
		ln, err := Listen(path)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			c, e := ln.AcceptUnix()
			if e != nil {
				done <- e
				return
			}
			done <- ServeConn(c, tc.uid)
		}()
		c, err := net.Dial("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = c.Write([]byte(tc.payload + "\n"))
		var r Reply
		readErr := json.NewDecoder(c).Decode(&r)
		c.Close()
		err = <-done
		ln.Close()
		if tc.want {
			if err != nil || readErr != nil || !r.OK {
				t.Fatalf("ping denied %v %v", err, readErr)
			}
		} else if err == nil {
			t.Fatal("bad agent operation allowed")
		}
	}
}
