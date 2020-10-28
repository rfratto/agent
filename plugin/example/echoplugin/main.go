package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/grafana/agent/plugin/conn"
	"github.com/grafana/agent/plugin/example/echoproto"
	"google.golang.org/grpc"
)

func main() {
	srv := grpc.NewServer()
	echoproto.RegisterEchoServer(srv, Echo{})

	var rwc io.ReadWriteCloser = &conn.RWC{
		Reader: os.Stdin,
		Writer: os.Stdout,
		Closer: ioutil.NopCloser(nil),
	}

	// TODO(rfratto): set to a ioconn
	var lis net.Listener = conn.NewListener(rwc)
	if err := srv.Serve(lis); err != nil {
		log.Println(err)
	}
}

type Echo struct {
	Prefix string
}

func (e Echo) Echo(_ context.Context, msg *echoproto.EchoMessage) (*echoproto.EchoMessage, error) {
	return &echoproto.EchoMessage{
		Text: e.Prefix + msg.Text,
	}, nil
}
