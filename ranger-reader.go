package main

import (
	"bytes"
	"io"

	"github.com/minio/minio-go/v7"
)

type Range struct {
	Enabled bool
	Start   int64
	End     int64
}

type RangeReader struct {
	ContentSize int64
	Range       Range
	Handle      *minio.Object
	index       int64
}

func (r *RangeReader) Close() error {
	return r.Handle.Close()
}

func (r *RangeReader) Read(p []byte) (n int, err error) {
	if r.index > r.Range.End {
		return 0, io.EOF
	}

	if r.index == 0 && r.Range.Start > 0 {
		r.index = r.Range.Start
	}

	if int64(len(p)) <= (r.Range.End - r.index + int64(1)) {
		read, readErr := r.Handle.ReadAt(p, r.index)
		r.index = r.index + int64(read)
		return read, readErr
	}

	p2 := make([]byte, r.Range.End - r.index + int64(1))
	read, readErr := r.Handle.ReadAt(p2, r.index)
	if readErr != nil {
		return 0, readErr
	}

	buf := bytes.NewBuffer(p2)
	read, readErr = buf.Read(p)
	r.index = r.index + int64(read)
	return read, readErr
}