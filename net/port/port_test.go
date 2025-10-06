package port

import (
	"fmt"
	"net"
	"testing"
)

func TestFree(t *testing.T) {
	p, err := Free()
	if err != nil {
		t.Fatal(err)
	}

	if p == 0 {
		t.Fatal("port.Free() returned 0")
	}

	var cfg net.ListenConfig

	l, err := cfg.Listen(t.Context(), "tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
}
