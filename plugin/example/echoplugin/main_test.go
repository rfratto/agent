package main

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/grafana/agent/plugin/conn"
	"github.com/grafana/agent/plugin/example/echoproto"
	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestIOServer(t *testing.T) {
	var (
		peerA = grpc.NewServer()
		peerB = grpc.NewServer()
	)
	echoproto.RegisterEchoServer(peerA, Echo{Prefix: "Peer A: "})
	echoproto.RegisterEchoServer(peerB, Echo{Prefix: "Peer B: "})

	// Create two io.Pipes. data written to peerAData is read by peerBData, and vice-versa.
	peerAData, peerBData := peerConns()

	peerASession, err := yamux.Server(peerAData, yamux.DefaultConfig())
	require.NoError(t, err)

	peerBSession, err := yamux.Client(peerBData, yamux.DefaultConfig())
	require.NoError(t, err)

	go peerA.Serve(peerASession)
	go peerB.Serve(peerBSession)

	peerABConn, err := grpc.Dial("stdout-ab", grpc.WithInsecure(), sessionDialer(peerASession))
	require.NoError(t, err)

	peerBAConn, err := grpc.Dial("stdout-ab", grpc.WithInsecure(), sessionDialer(peerBSession))
	require.NoError(t, err)

	peerACli := echoproto.NewEchoClient(peerABConn)
	peerBCli := echoproto.NewEchoClient(peerBAConn)

	resp, err := peerACli.Echo(context.Background(), &echoproto.EchoMessage{Text: "Hello from peer A!"})
	require.NoError(t, err)
	require.Equal(t, "Peer B: Hello from peer A!", resp.Text)

	resp, err = peerBCli.Echo(context.Background(), &echoproto.EchoMessage{Text: "Hello from peer B!"})
	require.NoError(t, err)
	require.Equal(t, "Peer A: Hello from peer B!", resp.Text)
}

func peerConns() (peerA, peerB io.ReadWriteCloser) {
	peerAReader, peerBWriter := io.Pipe()
	peerBReader, peerAWriter := io.Pipe()

	var (
		rwcA = &conn.RWC{
			Reader: peerAReader,
			Closer: peerAReader,
			Writer: peerAWriter,
		}
		rwcB = &conn.RWC{
			Reader: peerBReader,
			Closer: peerBReader,
			Writer: peerBWriter,
		}
	)

	return rwcA, rwcB
}

func sessionDialer(sess *yamux.Session) grpc.DialOption {
	return grpc.WithContextDialer(func(c context.Context, s string) (net.Conn, error) {
		return sess.Open()
	})
}
