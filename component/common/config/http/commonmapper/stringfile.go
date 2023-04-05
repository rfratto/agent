package commonmapper

import (
	"io/fs"
	"strings"
	"time"

	"go.uber.org/atomic"
)

type stringFile struct {
	closed      atomic.Bool
	valueReader *strings.Reader

	fi *constFileInfo
}

var _ fs.File = (*stringFile)(nil)

func newStringFile(name, value string) *stringFile {
	sr := strings.NewReader(value)
	sr.Len()

	return &stringFile{
		valueReader: sr,
		fi: &constFileInfo{
			name:    name,
			size:    int64(sr.Len()),
			mode:    0400, // Read only by owner
			modTime: time.Now(),
		},
	}
}

func (sf *stringFile) Stat() (fs.FileInfo, error) {
	return sf.fi, nil
}

func (sf *stringFile) Read(b []byte) (int, error) {
	if sf.closed.Load() {
		return 0, fs.ErrClosed
	}

	return sf.valueReader.Read(b)
}

func (sf *stringFile) Close() error {
	if !sf.closed.CAS(false, true) {
		return fs.ErrClosed
	}
	return nil
}

type constFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

var _ fs.FileInfo = (*constFileInfo)(nil)

func (fi *constFileInfo) Name() string       { return fi.name }
func (fi *constFileInfo) Size() int64        { return fi.size }
func (fi *constFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *constFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *constFileInfo) IsDir() bool        { return false }
func (fi *constFileInfo) Sys() any           { return nil }
