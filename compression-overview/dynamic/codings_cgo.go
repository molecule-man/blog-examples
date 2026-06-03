//go:build cgo

package main

import (
	"io"

	"github.com/DataDog/zstd"
	"github.com/google/brotli/go/cbrotli"
)

var baseCodings = []coding{
	brStreaming, // swap to brOneShot if needed
	streaming("zstd", 3, 1, 22, func(dst io.Writer, level int) (io.WriteCloser, error) {
		return zstd.NewWriterLevel(dst, level), nil
	}),
	gzipCoding,
}

var brStreaming = streaming("br", 6, 0, 11, func(dst io.Writer, level int) (io.WriteCloser, error) {
	return cbrotli.NewWriter(dst, cbrotli.WriterOptions{Quality: level}), nil
})

var brOneShot = oneShot("br", 6, 0, 11, func(data []byte, level int) ([]byte, error) {
	return cbrotli.Encode(data, cbrotli.WriterOptions{Quality: level})
})
