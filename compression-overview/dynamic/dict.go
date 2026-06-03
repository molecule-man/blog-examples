package main

import (
	"crypto/sha256"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/molecule-man/go-brrr"
)

func withPrefix(prefix []byte, c coding) coding {
	inner := c.encode
	c.encode = func(dst io.Writer, src io.Reader, level int) error {
		if _, err := dst.Write(prefix); err != nil {
			return err
		}
		return inner(dst, src, level)
	}
	return c
}

func dictCodings(dict []byte) ([]coding, error) {
	sum := sha256.Sum256(dict)

	pd, err := brrr.PrepareDictionary(dict)
	if err != nil {
		return nil, err
	}

	// dcb header: 0xff 'D' 'C' 'B', then the 32-byte dictionary hash, then a
	// brotli stream that references the dictionary. go-brrr compound
	// dictionaries require level >= 2.
	dcbHeader := append([]byte{0xff, 'D', 'C', 'B'}, sum[:]...)
	dcb := withPrefix(dcbHeader, pooledStreaming("dcb", 6, 2, 11, func(dst io.Writer, level int) (resettable, error) {
		return brrr.NewWriterOptions(dst, level, brrr.WriterOptions{
			Dictionaries: []*brrr.PreparedDictionary{pd},
		})
	}))

	// dcz header: a zstd skippable frame (magic 0x184D2A5E, content length 32)
	// carrying the dictionary hash, then a zstd frame that uses the dictionary
	// as a raw dictionary (id 0, so no Dictionary_ID is written into the frame).
	dczHeader := append([]byte{0x5e, 0x2a, 0x4d, 0x18, 0x20, 0x00, 0x00, 0x00}, sum[:]...)
	dcz := withPrefix(dczHeader, pooledStreaming("dcz", 3, 1, 22, func(dst io.Writer, level int) (resettable, error) {
		return zstd.NewWriter(dst,
			zstd.WithEncoderDictRaw(0, dict),
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
			zstd.WithEncoderConcurrency(1))
	}))

	return []coding{dcb, dcz}, nil
}
