package conn

import (
	"context"
	"io"
	"net"

	"google.golang.org/grpc"
)

type RWC struct {
	io.Reader
	io.Writer
	io.Closer
}

func Dialer(tc *TextConn) grpc.DialOption {
	return grpc.WithContextDialer(func(c context.Context, s string) (net.Conn, error) {
		return tc, nil
	})
}
