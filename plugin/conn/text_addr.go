package conn

// TextAddr implements net.Addr for net.Conns using TextConn.
type TextAddr struct {
	// ID that represents the local- or remote-site connection.
	ID string
}

// Network implements net.Addr.
func (a TextAddr) Network() string { return "plugin" }

// String implements net.Addr.
func (a TextAddr) String() string { return a.ID }
