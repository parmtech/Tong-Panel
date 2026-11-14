//go:build linux

package agent

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzDecodeRequestOnlyAllowsPing(f *testing.F) {
	for _, seed := range []string{
		`{"op":"ping"}` + "\n",
		`{"op":"root.exec"}` + "\n",
		`{"op":"ping","cmd":"whoami"}` + "\n",
		`{"op":"root.exec","op":"ping"}` + "\n",
		`{"op":"ping"}{"op":"other"}` + "\n",
		`{"op":"ping"} trailing` + "\n",
		`{` + "\n",
		strings.Repeat("x", 513) + "\n",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, frame []byte) {
		request, err := decodeRequest(frame)
		if err == nil && request.Op != "ping" {
			t.Fatalf("decoder accepted non-ping operation %q", request.Op)
		}
	})
}

func TestDecodeRequestRejectsDuplicateOperation(t *testing.T) {
	if request, err := decodeRequest([]byte(`{"op":"root.exec","op":"ping"}` + "\n")); err == nil {
		t.Fatalf("duplicate operation key accepted as %q", request.Op)
	}
}

func TestOnlyPingAndPeerUID(t *testing.T) {
	for _, tc := range []struct {
		payload string
		uid     uint32
		want    bool
	}{{`{"op":"ping"}`, uint32(os.Getuid()), true}, {`{"op":"root.exec"}`, uint32(os.Getuid()), false}, {`{"op":"ping","cmd":"whoami"}`, uint32(os.Getuid()), false}, {`{"op":"ping"}`, uint32(os.Getuid() + 1000), false}} {
		path := filepath.Join(t.TempDir(), "agent.sock")
		if err := os.Chmod(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
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

func TestListenRejectsSharedParentAndSymlink(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(filepath.Join(parent, "agent.sock")); err == nil {
		t.Fatal("accepted group-readable socket parent")
	}
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(parent, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(filepath.Join(link, "agent.sock")); err == nil {
		t.Fatal("accepted symlink parent")
	}
	ln, err := Listen(filepath.Join(parent, "agent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	st, err := os.Stat(filepath.Join(parent, "agent.sock"))
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("insecure socket mode %v %v", st, err)
	}
}

func TestAgentRejectsExtraJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.sock")
	if err := os.Chmod(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	ln, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan error, 1)
	go func() {
		c, e := ln.AcceptUnix()
		if e != nil {
			done <- e
			return
		}
		done <- ServeConn(c, uint32(os.Getuid()))
	}()
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.Write([]byte("{\"op\":\"ping\"}{\"op\":\"other\"}\n"))
	_ = c.Close()
	if err := <-done; err == nil {
		t.Fatal("accepted concatenated JSON")
	}
}
