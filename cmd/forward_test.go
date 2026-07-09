package cmd

import (
	"net"
	"strconv"
	"testing"
)

// TestFindFreePort locks that findFreePort returns a bindable localhost TCP port.
// It asks for a free port, then listens on that port to confirm it was released
// and is usable for the SSH local forward.
func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() error = %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("findFreePort() port = %d, want a valid TCP port", port)
	}

	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("findFreePort() returned port %d that cannot be bound: %v", port, err)
	}
	defer func() { _ = l.Close() }()
}
