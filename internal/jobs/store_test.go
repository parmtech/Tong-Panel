package jobs

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStorePersistenceIsolationAndIdempotency(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secure", "jobs.json")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Create("a", KindNoop, "my-request-001")
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := s.Create("a", KindNoop, "my-request-001")
	if err != nil || repeat.ID != a.ID {
		t.Fatal("duplicate created")
	}
	if _, err = s.Get("b", a.ID); err != ErrNotFound {
		t.Fatalf("leaked tenant: %v", err)
	}
	if len(s.List("b")) != 0 {
		t.Fatal("list leaked tenant")
	}
	if _, err = s.Transition("a", a.ID, Done); err == nil {
		t.Fatal("invalid transition accepted")
	}
	if _, err = s.Transition("a", a.ID, Running); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Transition("a", a.ID, Done); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := restored.Get("a", a.ID)
	if err != nil || b.State != Done {
		t.Fatalf("restart lost state: %#v %v", b, err)
	}
}
func TestConcurrentDuplicate(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "q.json"))
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, err := s.Create("x", KindNoop, "one-request-id")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- j.ID
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatal("duplicate race")
		}
	}
	if len(s.List("x")) != 1 {
		t.Fatal("extra job recorded")
	}
}
func TestCorruptStoreFailsClosed(t *testing.T) {
	p := filepath.Join(t.TempDir(), "q.json")
	if err := os.WriteFile(p, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); err == nil {
		t.Fatal("bad journal accepted")
	}
}
