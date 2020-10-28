// Package ioconn implements net.Conn, net.Addr, and net.Listener using
// io.ReadWriteClosers.
package ioconn

import "net"

// Listener impements net.Listener, multiplexing multiple Conns over a single
// io.ReadWriteCloser.
type Listener struct {
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	return nil, nil
}

// Close closes the listener.
func (l *Listener) Close() error {
	return nil
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return nil
}

// Let's backpedal. We should be using gRPC for all the communication here and
// not put anything in front of gRPC. This allows any plugin to be written.
//
// So, in that case, we have different use cases:
//
// 1. The host invokes a plugin's RPC directly
// 2. The plugin sends a generic gRPC message to the host's proxy system.
// 3. The plugin sends a generic gRPC message to the plugin.
//
// One thing a plugin won't be able to know is where a gRPC message is actually coming from.
// All it can tell is that there was a message from the host or it sent a message to a host.
//
// I think this is going to depend on what we really want to do here. Do we want these plugins to
// act as if there was a real network when all they're using is gRPC? Or do we want these plugins
// to use a special set of messages to communicate with one another?
//
// We could build ioconn on top of these special messages so Go's way of handling it is extra nice.
// I think that's probably the approach we want to go for.
//
// Based on that, here's the deal:
// 1. A bidi communication channel for sending generic gRPC messages between the host and plugin.
//    Both sides should interpret these as normal incoming messages to be handled.
// 2. A separate set of RPC methods.
//
// Now, then, "Dialing" and getting a *grpc.ClientConn doesn't actually do
// much. The underlying net.Conn just writes those normal gRPC messages out.
