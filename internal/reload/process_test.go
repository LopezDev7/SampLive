package reload

import "testing"

func TestParseNetstat(t *testing.T) {
	const netstat = `
Active Connections

  Proto  Local Address          Foreign Address        State           PID
  TCP    0.0.0.0:7777           0.0.0.0:0              LISTENING       4321
  TCP    192.168.1.5:7777       0.0.0.0:0              LISTENING       9999
  UDP    0.0.0.0:7777           *:*                                    9000
  UDP    0.0.0.0:7778           *:*                                    1111
  TCP    0.0.0.0:80             0.0.0.0:0              LISTENING       4
`

	// TCP listener is preferred and returned first.
	pid, err := parseNetstat([]byte(netstat), 7777)
	if err != nil {
		t.Fatalf("parseNetstat returned error: %v", err)
	}
	if pid != 4321 {
		t.Fatalf("expected TCP pid 4321, got %d", pid)
	}

	// UDP-only server (open.mp) is found when no TCP listener exists.
	udpOnly := `
  TCP    0.0.0.0:80   0.0.0.0:0   LISTENING   4
  UDP    0.0.0.0:7799 *:*                     5555
`
	pid, err = parseNetstat([]byte(udpOnly), 7799)
	if err != nil {
		t.Fatalf("parseNetstat returned error: %v", err)
	}
	if pid != 5555 {
		t.Fatalf("expected UDP pid 5555, got %d", pid)
	}

	// Nothing on the port: 0, no error.
	pid, err = parseNetstat([]byte(netstat), 4444)
	if err != nil {
		t.Fatalf("parseNetstat returned error: %v", err)
	}
	if pid != 0 {
		t.Fatalf("expected 0, got %d", pid)
	}
}
