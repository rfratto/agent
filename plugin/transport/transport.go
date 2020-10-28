// Package transport implements net.Conn, net.Listener, and net.Addr using
// a bidi gRPC stream. This allows to use gRPC for carrying gRPC messages.
package transport

import (
	"io"
	"net"
	"time"
)

// Listener implements net.Listener for a MessageStream. It only ever returns
// one Conn, which represents the stream itself.
type Listener struct {
	ch chan net.Conn
}

// NewListener creates a new Listener from a stream and a mechanism to use to
// close it.
func NewListener(stream MessageStream, onClose func() error) *Listener {
	ch := make(chan net.Conn, 1)
	ch <- NewConn(stream, onClose)
	return &Listener{ch: ch}
}

// Accept returns the stream as a Conn. Accept will not return again until the
// Listener is closed.
func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.ch
	if !ok {
		return nil, io.EOF
	}
	return conn, nil
}

// Close closes the listener and unblocks any calls to Accept.
func (l *Listener) Close() error {
	close(l.ch)
	return nil
}

func (l *Listener) Addr() net.Addr {
	// TODO(rfratto): handle this
	return nil
}

// MessageStream is implemented by both Transport_InitiateClient and Transport_InitiateServer.
type MessageStream interface {
	Send(*TransportMessage) error
	Recv() (*TransportMessage, error)
}

// Conn implements net.Conn using a MessageStream.
type Conn struct {
	stream  MessageStream
	onClose func() error

	// Unread bytes to fill to calls to b before reading from stream again.
	unread []byte
}

// NewConn creates a new connection from a stream.
func NewConn(stream MessageStream, onClose func() error) *Conn {
	return &Conn{
		stream:  stream,
		onClose: onClose,
	}
}

// Read reads from the MessageStream into b. If b is not big enough to contain the full
// message, the next call to Read will be filled with the remaining bytes.
func (c *Conn) Read(b []byte) (n int, err error) {
	// Read from the unused bytes first.
	if len(c.unread) > 0 {
		n = copy(b, c.unread)
		c.unread = c.unread[n:]
	}

	// If there's still more room in b beyond what we've written, we can read in a new packet.
	if len(b) > n {
		msg, err := c.stream.Recv()
		if err != nil {
			return n, err
		}

		sz := copy(b[n:], msg.Data)
		c.unread = msg.Data[sz:]

		n += sz
	}

	return
}

// Write writes b to the MessageStream.
func (c *Conn) Write(b []byte) (n int, err error) {
	err = c.stream.Send(&TransportMessage{Data: b})
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes a Conn.
func (c *Conn) Close() error {
	if c.onClose != nil {
		return c.onClose()
	}
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	// TODO(rfrato): NYI
	return nil
}

func (c *Conn) RemoteAddr() net.Addr {
	// TODO(rfrato): NYI
	return nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	// TODO(rfrato): NYI
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	// TODO(rfrato): NYI
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	// TODO(rfrato): NYI
	return nil
}
