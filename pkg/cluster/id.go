package cluster

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/rfratto/croissant/id"
)

// keyGenerator generates a key given an input string.
type keyGenerator struct {
	size int
}

func NewKeyGenerator(size int) id.Generator {
	return keyGenerator{size: size}
}

// Get gets a new ID based on s.
func (kg keyGenerator) Get(s string) id.ID {
	h := md5.New()
	fmt.Fprint(h, s)

	sum := h.Sum(nil)
	var (
		high = binary.BigEndian.Uint64(sum[:8])
		low  = binary.BigEndian.Uint64(sum[8:])
	)

	if kg.size != 128 {
		low = high ^ low
		high = 0
	}
	low = low % id.MaxForSize(kg.size).Low

	return id.ID{High: high, Low: low}
}
