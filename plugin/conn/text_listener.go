package conn

import (
	"io"
	"net"
)

type TextListener struct {
	ch chan net.Conn
}

func NewListener(rwc io.ReadWriteCloser) *TextListener {
	ch := make(chan net.Conn, 1)
	ch <- &TextConn{
		ReadWriteCloser: rwc,
		LocalTextAddr:   &TextAddr{ID: "local"},
		RemoteTextAddr:  &TextAddr{ID: "remote"},
	}
	return &TextListener{ch: ch}
}

// Accept returns the stream as a Conn. Accept will not return again until the
// Listener is closed.
func (l *TextListener) Accept() (net.Conn, error) {
	conn, ok := <-l.ch
	if !ok {
		return nil, io.EOF
	}
	return conn, nil
}

// Close closes the listener and unblocks any calls to Accept.
func (l *TextListener) Close() error {
	close(l.ch)
	return nil
}

func (l *TextListener) Addr() net.Addr {
	return &TextAddr{ID: "local"}
}
