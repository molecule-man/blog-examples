// Toy SSE server that streams the files of a directory to connected clients as
// brotli-compressed events.
// ATTENTION: this is a demo, not production code. There is no dictionary
// negotiation, auth, or backpressure handling.
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"blog-examples/compression-overview/streaming/proto"

	"github.com/molecule-man/go-brrr"
)

const (
	defaultLevel = 5
	defaultDelay = 200 * time.Millisecond
)

type namedFile struct {
	name string
	data []byte
}

type server struct {
	dir  string
	dict []byte
	pd   *brrr.PreparedDictionary
}

func (s *server) files() ([]namedFile, error) {
	entries, err := os.ReadDir(s.dir) // ReadDir returns entries sorted by name
	if err != nil {
		return nil, err
	}
	var out []namedFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, namedFile{name: e.Name(), data: data})
	}
	return out, nil
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = proto.ModeReset
	}
	if !proto.ValidMode(mode) {
		http.Error(w, "unknown mode "+strconv.Quote(mode), http.StatusBadRequest)
		return
	}
	level := defaultLevel
	if v := r.URL.Query().Get("level"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			level = n
		}
	}
	delay := defaultDelay
	if v := r.URL.Query().Get("delay"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			delay = d
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	files, err := s.files()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set(proto.ModeHeader, mode)
	w.WriteHeader(http.StatusOK)

	log.Printf("client %s: mode=%s level=%d files=%d", r.RemoteAddr, mode, level, len(files))

	var total int
	if proto.Independent(mode) {
		total, err = s.streamIndependent(w, flusher, mode, level, delay, files)
	} else {
		total, err = s.streamContinuous(w, flusher, level, delay, files)
	}
	if err != nil {
		// Most likely the client disconnected mid-stream; nothing to recover.
		log.Printf("client %s: %v", r.RemoteAddr, err)
		return
	}

	fmt.Fprintf(w, "event: done\ndata: %d\n\n", total)
	flusher.Flush()
}

func (s *server) streamIndependent(w io.Writer, f http.Flusher, mode string, level int, delay time.Duration, files []namedFile) (int, error) {
	var buf bytes.Buffer
	var bw *brrr.Writer
	var err error
	if mode == proto.ModeDict {
		bw, err = brrr.NewWriterOptions(&buf, level, brrr.WriterOptions{
			Dictionaries: []*brrr.PreparedDictionary{s.pd},
		})
	} else {
		bw, err = brrr.NewWriter(&buf, level)
	}
	if err != nil {
		return 0, err
	}

	total := 0
	for i, file := range files {
		buf.Reset()
		bw.Reset(&buf)
		if err := proto.WriteRecord(bw, file.data); err != nil {
			return total, err
		}
		if err := bw.Close(); err != nil {
			return total, err
		}
		total += buf.Len()
		if err := sendEvent(w, f, i, buf.Bytes()); err != nil {
			return total, err
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return total, nil
}

func (s *server) streamContinuous(w io.Writer, f http.Flusher, level int, delay time.Duration, files []namedFile) (int, error) {
	var buf bytes.Buffer
	bw, err := brrr.NewWriter(&buf, level)
	if err != nil {
		return 0, err
	}

	total := 0
	// drain emits whatever the encoder has written to buf so far as one frame.
	// Resetting buf truncates the wire buffer only; the encoder's window state
	// lives in bw and is untouched.
	drain := func(id int) error {
		chunk := buf.Bytes()
		total += len(chunk)
		err := sendEvent(w, f, id, chunk)
		buf.Reset()
		return err
	}

	for i, file := range files {
		if err := proto.WriteRecord(bw, file.data); err != nil {
			return total, err
		}
		// Flush ends a block without finalizing the stream, so this event's
		// bytes reach the client now rather than waiting for the buffer to fill.
		if err := bw.Flush(); err != nil {
			return total, err
		}
		if err := drain(i); err != nil {
			return total, err
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	// Close finalizes the single stream; emit the final empty block as a last frame.
	if err := bw.Close(); err != nil {
		return total, err
	}
	if buf.Len() > 0 {
		if err := drain(len(files)); err != nil {
			return total, err
		}
	}
	return total, nil
}

func sendEvent(w io.Writer, f http.Flusher, id int, chunk []byte) error {
	if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", id, base64.StdEncoding.EncodeToString(chunk)); err != nil {
		return err
	}
	f.Flush()
	return nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s <port> <dir> <dict-file>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}
	port, dir, dictPath := os.Args[1], os.Args[2], os.Args[3]

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "dir %q: not a directory\n", dir)
		os.Exit(1)
	}
	dict, err := os.ReadFile(dictPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dictionary %q: %v\n", dictPath, err)
		os.Exit(1)
	}
	pd, err := brrr.PrepareDictionary(dict)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dictionary %q: %v\n", dictPath, err)
		os.Exit(1)
	}

	s := &server{dir: dir, dict: dict, pd: pd}
	http.HandleFunc("/events", s.handle)

	log.Printf("serving %s on :%s — GET /events?mode={reset,dict,stream}", dir, port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
