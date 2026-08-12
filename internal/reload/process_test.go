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

func TestParseSS(t *testing.T) {
	// SA-MP: a TCP listener on the game port.
	tcp := `
State      Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
LISTEN     0      128    0.0.0.0:7777        0.0.0.0:*        users:(("samp-server",pid=4321,fd=3))
LISTEN     0      128    0.0.0.0:22          0.0.0.0:*        users:(("sshd",pid=1,fd=3))
`
	pid, err := parseSS([]byte(tcp), 7777, false)
	if err != nil {
		t.Fatalf("parseSS returned error: %v", err)
	}
	if pid != 4321 {
		t.Fatalf("expected TCP pid 4321, got %d", pid)
	}

	// open.mp: a UDP-only socket. UDP rows have no LISTEN state.
	udp := `
State      Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
UNCONNECTED 0    0      0.0.0.0:7777        0.0.0.0:*        users:(("omp-server",pid=5555,fd=4))
`
	pid, err = parseSS([]byte(udp), 7777, true)
	if err != nil {
		t.Fatalf("parseSS (udp) returned error: %v", err)
	}
	if pid != 5555 {
		t.Fatalf("expected UDP pid 5555, got %d", pid)
	}

	// The same UDP socket must not be found when scanning TCP only.
	if pid, err := parseSS([]byte(udp), 7777, false); err != nil {
		t.Fatalf("parseSS (tcp) returned error: %v", err)
	} else if pid != 0 {
		t.Fatalf("expected 0 for a UDP socket in TCP scan, got %d", pid)
	}
}

func TestParseNetstatLinux(t *testing.T) {
	tcp := `
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
tcp   0      0      0.0.0.0:7777            0.0.0.0:*               LISTEN      4321/samp-server
`
	pid, err := parseNetstatLinux([]byte(tcp), 7777, false)
	if err != nil {
		t.Fatalf("parseNetstatLinux returned error: %v", err)
	}
	if pid != 4321 {
		t.Fatalf("expected TCP pid 4321, got %d", pid)
	}

	// open.mp: UDP rows have no state column; the last field is still PID/Name.
	udp := `
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
udp   0      0      0.0.0.0:7777            0.0.0.0:*                           5555/omp-server
`
	pid, err = parseNetstatLinux([]byte(udp), 7777, true)
	if err != nil {
		t.Fatalf("parseNetstatLinux (udp) returned error: %v", err)
	}
	if pid != 5555 {
		t.Fatalf("expected UDP pid 5555, got %d", pid)
	}

	// Port in use but PID hidden (not running as the server user).
	hidden := `
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
udp   0      0      0.0.0.0:7777            0.0.0.0:*                           -
`
	if _, err := parseNetstatLinux([]byte(hidden), 7777, true); err == nil {
		t.Fatal("expected an error when the PID is not visible")
	}
}
