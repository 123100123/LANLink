package discovery

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestBroadcastTargetsIncludesGlobal(t *testing.T) {
	targets := broadcastTargets(8788)
	found := false
	for _, tgt := range targets {
		if tgt.IP.Equal(net.IPv4bcast) && tgt.Port == 8788 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the global broadcast address among targets")
	}
}

func TestScanReturnsEmptyOnTimeout(t *testing.T) {
	// Bind an ephemeral port that nothing announces to; Scan should return a
	// non-nil empty slice when the deadline passes.
	hosts, err := Scan(150*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if hosts == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(hosts) != 0 {
		t.Fatalf("expected no hosts, got %d", len(hosts))
	}
}

func TestAnnouncementRoundTrips(t *testing.T) {
	in := Announcement{
		Service: ServiceName,
		Name:    "host",
		Addr:    "192.168.1.5:8787",
		Port:    "8787",
		Version: "0.5.0",
		Open:    true,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Announcement
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: %+v != %+v", out, in)
	}
}
