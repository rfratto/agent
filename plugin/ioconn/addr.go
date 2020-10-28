package ioconn

import "fmt"

// Addr implements net.Addr for ioconn connections.
type Addr uint64

// Network implements net.Addr.
func (a Addr) Network() string { return "ioconn" }

// String implements net.Addr.
func (a Addr) String() string { return fmt.Sprintf("%d", a) }
