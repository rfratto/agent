package conn

import (
	"context"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var _ net.Conn = (*TextConn)(nil)

// TextConn implements net.Conn over a ReadWriteCloser.
type TextConn struct {
	io.ReadWriteCloser

	LocalTextAddr  *TextAddr
	RemoteTextAddr *TextAddr

	mtx           sync.Mutex
	readDeadline  time.Time
	writeDeadline time.Time
}

// Read implements net.Conn.Read with support for read deadlines.
func (c *TextConn) Read(p []byte) (n int, err error) {
	c.mtx.Lock()
	deadline := c.readDeadline
	c.mtx.Unlock()

	// Short-circuit: if there's no deadline just pass through to the underlying
	// RWC. Or if we're already past the deadline, return early.
	if deadline.IsZero() {
		return c.ReadWriteCloser.Read(p)
	} else if time.Now().After(deadline) {
		return 0, os.ErrDeadlineExceeded
	}

	ch := make(chan bool, 1)
	go func() {
		n, err = c.ReadWriteCloser.Read(p)
		close(ch)
	}()

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		return 0, os.ErrDeadlineExceeded
	case <-ch:
		return n, err
	}
}

// Write implements net.Conn.Write with support for write deadlines.
func (c *TextConn) Write(p []byte) (n int, err error) {
	c.mtx.Lock()
	deadline := c.writeDeadline
	c.mtx.Unlock()

	// Short-circuit: if there's no deadline just pass through to the underlying
	// RWC. Or if we're already past the deadline, return early.
	if deadline.IsZero() {
		return c.ReadWriteCloser.Write(p)
	} else if time.Now().After(deadline) {
		return 0, os.ErrDeadlineExceeded
	}

	ch := make(chan bool, 1)
	go func() {
		n, err = c.ReadWriteCloser.Write(p)
		close(ch)
	}()

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		return 0, os.ErrDeadlineExceeded
	case <-ch:
		return n, err
	}
}

// LocalAddr returns the local network address.
func (c *TextConn) LocalAddr() net.Addr {
	return c.LocalTextAddr
}

// RemoteAddr returns the remote network address.
func (c *TextConn) RemoteAddr() net.Addr {
	return c.RemoteTextAddr
}

// SetDeadline implements net.Conn.
func (c *TextConn) SetDeadline(t time.Time) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

// SetReadDeadline implements net.Conn.
func (c *TextConn) SetReadDeadline(t time.Time) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.readDeadline = t
	return nil
}

// SetWriteDeadline implements net.Conn.
func (c *TextConn) SetWriteDeadline(t time.Time) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.writeDeadline = t
	return nil
}
