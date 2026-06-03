package proto

import (
	"encoding/binary"
	"io"
)

const ModeHeader = "X-Brrr-Mode"

const (
	ModeReset  = "reset"  // compress every event separately
	ModeDict   = "dict"   // same as reset but with a shared dictionary
	ModeStream = "stream" // no resets, one continuous stream across all events
)

func Independent(mode string) bool {
	return mode == ModeReset || mode == ModeDict
}

func ValidMode(mode string) bool {
	return mode == ModeReset || mode == ModeDict || mode == ModeStream
}

func WriteRecord(w io.Writer, payload []byte) error {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func ReadRecord(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	payload := make([]byte, binary.BigEndian.Uint32(hdr[:]))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
