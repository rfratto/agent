package ioconn

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// A Packet is an individual message transmitted over io.
//
// The packet is marshaled in the following form:
//
// uvarint RemoteAddr
// uvarint LocalAddr
// uvarint len(Data)
// [data]
type Packet struct {
	// RemoteAddr is the Addr for where the packet was sent.
	RemoteAddr Addr

	// LocalAddr is the Addr for where the packet is being received.
	LocalAddr Addr

	// Data is the data of the packet.
	Data []byte
}

// Unmarshal reads the packet from r and fills p.
func (p *Packet) Unmarshal(r io.Reader) error {
	br := bufio.NewReaderSize(r, 1)

	val, err := binary.ReadUvarint(br)
	if err != nil {
		return err
	}
	p.RemoteAddr = Addr(val)

	val, err = binary.ReadUvarint(br)
	if err != nil {
		return err
	}
	p.LocalAddr = Addr(val)

	dataSz, err := binary.ReadUvarint(br)
	if err != nil {
		return err
	}
	data := make([]byte, dataSz)
	n, err := r.Read(data)
	if err != nil {
		return err
	} else if uint64(n) != dataSz {
		return fmt.Errorf("incomplete read")
	}

	p.Data = data
	return nil
}

// Marshal writes the packet to w.
func (p Packet) Marshal(w io.Writer) error {
	buf := make([]byte, binary.MaxVarintLen64)

	writeNumbers := []uint64{
		uint64(p.RemoteAddr),
		uint64(p.LocalAddr),
		uint64(len(p.Data)),
	}
	for _, value := range writeNumbers {
		n := binary.PutUvarint(buf, value)
		if sz, err := w.Write(buf[:n]); err != nil {
			return err
		} else if sz != n {
			return fmt.Errorf("partial write")
		}
	}

	sz, err := io.Copy(w, bytes.NewReader(p.Data))
	if err != nil {
		return err
	} else if int(sz) != len(p.Data) {
		return fmt.Errorf("partial write")
	}

	return nil
}
