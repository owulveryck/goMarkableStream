package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc/credentials"
)

type callInfoAuthenticator struct {
	tlsCreds credentials.TransportCredentials
}

// ClientHandshake does the authentication handshake specified by the
// corresponding authentication protocol on rawConn for clients. It returns
// the authenticated connection and the corresponding auth information
// about the connection.  The auth information should embed CommonAuthInfo
// to return additional information about the credentials. Implementations
// must use the provided context to implement timely cancellation.  gRPC
// will try to reconnect if the error returned is a temporary error
// (io.EOF, context.DeadlineExceeded or err.Temporary() == true).  If the
// returned error is a wrapper error, implementations should make sure that
// the error implements Temporary() to have the correct retry behaviors.
// Additionally, ClientHandshakeInfo data will be available via the context
// passed to this call.
//
// If the returned net.Conn is closed, it MUST close the net.Conn provided.
func (c *callInfoAuthenticator) ClientHandshake(ctx context.Context, s string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return c.tlsCreds.ClientHandshake(ctx, s, conn)
}

// ServerHandshake does the authentication handshake for servers. It returns
// the authenticated connection and the corresponding auth information about
// the connection. The auth information should embed CommonAuthInfo to return additional information
// about the credentials.
//
// If the returned net.Conn is closed, it MUST close the net.Conn provided.
func (c *callInfoAuthenticator) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	log.Println("New connection from: ", rawConn.RemoteAddr())
	callInfo := CallAuthInfo{RemoteAddr: rawConn.RemoteAddr()}
	if c.tlsCreds != nil {
		conn, info, err := c.tlsCreds.ServerHandshake(rawConn)
		if err != nil {
			return nil, nil, err
		}
		callInfo.Parent = info
		return conn, callInfo, nil
	}
	return rawConn, callInfo, nil
}

// Info provides the ProtocolInfo of this TransportCredentials.
func (c *callInfoAuthenticator) Info() credentials.ProtocolInfo {
	return c.tlsCreds.Info()
}

// Clone makes a copy of this TransportCredentials.
func (c *callInfoAuthenticator) Clone() credentials.TransportCredentials {
	return &callInfoAuthenticator{
		c.tlsCreds.Clone(),
	}
}

// OverrideServerName overrides the server name used to verify the hostname on the returned certificates from the server.
// gRPC internals also use it to override the virtual hosting name if it is set.
// It must be called before dialing. Currently, this is only used by grpclb.
func (c *callInfoAuthenticator) OverrideServerName(s string) error {
	return c.tlsCreds.OverrideServerName(s)
}

// CallAuthInfo as described here https://github.com/grpc/grpc-go/issues/334
type CallAuthInfo struct {
	RemoteAddr net.Addr
	Parent     credentials.AuthInfo
}

// AuthType is callinfo
func (t CallAuthInfo) AuthType() string {
	return "callinfo"
}
