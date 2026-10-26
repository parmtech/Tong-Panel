package preflight

import "testing"

func TestParseOSRelease(t *testing.T) {
	id, v, e := ParseOSRelease("NAME=Ubuntu\nID=ubuntu\nVERSION_ID=\"24.04\"\n")
	if e != nil || id != "ubuntu" || v != "24.04" {
		t.Fatalf("got %s %s %v", id, v, e)
	}
	if _, _, e = ParseOSRelease("invalid"); e == nil {
		t.Fatal("missing os fields accepted")
	}
}
