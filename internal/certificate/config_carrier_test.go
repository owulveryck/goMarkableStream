package certificate

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"sync"
	"testing"
)

func TestMake(t *testing.T) {
	//cw := NewCertWrapper(rand.Reader)
	t.Run("simple test", func(t *testing.T) {
		cw := NewCertConfigCarrier(rand.Reader)
		err := cw.Make()
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
		}(t, l)
		c, err := tls.Dial(ln.Addr().Network(), ln.Addr().String(), cw.ClientTLSConf)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		wg.Wait()
	})
	t.Run("two calls with random reader", func(t *testing.T) {
		cw := NewCertConfigCarrier(rand.Reader)
		err := cw.Make()
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
			if err == nil {
				t.Error("should have failed")
				return
			}
		}(t, l)
		cw2 := NewCertConfigCarrier(rand.Reader)
		err = cw.Make()
		if err != nil {
			t.Fatal(err)
		}
		_, err = tls.Dial(ln.Addr().Network(), ln.Addr().String(), cw2.ClientTLSConf)
		if err == nil {
			t.Fail()
		}
		//defer c.Close()
		wg.Wait()
	})
	t.Run("simple test with gob encoding", func(t *testing.T) {
		cw := NewCertConfigCarrier(rand.Reader)
		err := cw.Make()
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
		}(t, l)
		b, err := cw.GobEncode()
		if err != nil {
			t.Fatal(err)
		}
		var cw2 CertConfigCarrier
		err = cw2.GobDecode(b)
		if err != nil {
			t.Fatal(err)
		}
		c, err := tls.Dial(ln.Addr().Network(), ln.Addr().String(), cw2.ClientTLSConf)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		wg.Wait()
	})
}
