package port

import "net"

const proto = "tcp"
const addr = ":0"

// Free returns a free port on the local machine.
func Free() (int, error) {
	addr, err := net.ResolveTCPAddr(proto, addr)
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP(proto, addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
