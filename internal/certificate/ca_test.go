package certificate

import (
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"testing"
)

func TestCreateCertificate(t *testing.T) {
	cw := NewCertWrapper(rand.Reader)
	err := cw.CreateCertificate()
	if err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := tls.NewListener(ln, cw.ServerTLSConf)
	defer l.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func(t *testing.T, l net.Listener) {
		defer wg.Done()
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			t.Error("accept error", err)
			return
		}
		defer conn.Close()
		tlscon, ok := conn.(*tls.Conn)
		if !ok {
			t.Error("connexion is not a tls")
			return
		}
		err = tlscon.Handshake()
		if err != nil {
			t.Error("accept error", err)
			return
		}
		io.Copy(conn, conn)
	}(t, l)
	c, err := tls.Dial(ln.Addr().Network(), ln.Addr().String(), cw.ClientTLSConf)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	wg.Wait()
}
