package ioconn

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPacket(t *testing.T) {
	pkts := []Packet{
		{
			RemoteAddr: Addr(1234),
			LocalAddr:  Addr(5678),
			Data:       []byte("Hello, world!"),
		},
		{
			RemoteAddr: Addr(9012),
			LocalAddr:  Addr(3456),
			Data:       []byte("See you later!"),
		},
	}

	pr, pw := io.Pipe()
	go func() {
		for _, pkt := range pkts {
			err := pkt.Marshal(pw)
			require.NoError(t, err)
		}

		pw.Close()
	}()

	var gotPackets []Packet
	for i := 0; i < len(pkts); i++ {
		var pkt Packet
		err := pkt.Unmarshal(pr)
		require.NoError(t, err)
		gotPackets = append(gotPackets, pkt)
	}

	require.Equal(t, pkts, gotPackets)
}
