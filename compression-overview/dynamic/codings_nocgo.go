//go:build !cgo

package main

import (
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/molecule-man/go-brrr"
)

var baseCodings = []coding{
	pooledStreaming("br", 6, 0, 11, func(dst io.Writer, level int) (resettable, error) {
		return brrr.NewWriter(dst, level)
	}),
	pooledStreaming("zstd", 3, 1, 22, func(dst io.Writer, level int) (resettable, error) {
		return zstd.NewWriter(dst,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
			zstd.WithEncoderConcurrency(1))
	}),
	gzipCoding,
}
