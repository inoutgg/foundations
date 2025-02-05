package port

import (
	"fmt"
	"net"
)

const (
	proto = "tcp"
	addr  = ":0"
)

// Free returns a free port on the local machine.
func Free() (int, error) {
	addr, err := net.ResolveTCPAddr(proto, addr)
	if err != nil {
		return 0, fmt.Errorf("net/port: unable to resolve tcp addr: %w", err)
	}

	l, err := net.ListenTCP(proto, addr)
	if err != nil {
		return 0, fmt.Errorf("net/port: unable to listen tcp: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
