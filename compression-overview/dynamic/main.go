// Toy HTTP server that compresses a single file on the fly.
// ATTENTION: this is a demo, not production code! It serves only the purpose to
// showcase the compression ratio and speed. The dcz and dcb implementations are
// not fully correct as they omit the necessary dictionary negotiation and
// validation steps, so they should not be used as-is in production.
package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/gzip"
)

type coding struct {
	name   string
	def    int
	clamp  func(int) int
	encode func(dst io.Writer, src io.Reader, level int) error
}

func clampRange(lo, hi int) func(int) int {
	return func(n int) int {
		if n < lo {
			return lo
		}
		if n > hi {
			return hi
		}
		return n
	}
}

type resettable interface {
	io.WriteCloser
	Reset(dst io.Writer)
}

func pooledStreaming(name string, def, lo, hi int, newW func(io.Writer, int) (resettable, error)) coding {
	var mu sync.Mutex
	pools := map[int]*sync.Pool{}
	poolFor := func(level int) *sync.Pool {
		mu.Lock()
		defer mu.Unlock()
		if pools[level] == nil {
			pools[level] = &sync.Pool{}
		}
		return pools[level]
	}
	return coding{
		name: name, def: def, clamp: clampRange(lo, hi),
		encode: func(dst io.Writer, src io.Reader, level int) error {
			sp := poolFor(level)
			cw, _ := sp.Get().(resettable)
			if cw == nil {
				var err error
				if cw, err = newW(dst, level); err != nil {
					return err
				}
			} else {
				cw.Reset(dst)
			}
			defer func() {
				cw.Reset(io.Discard) // drop the reference to the response writer
				sp.Put(cw)
			}()
			if _, err := io.Copy(cw, src); err != nil {
				return err
			}
			return cw.Close()
		},
	}
}

func streaming(name string, def, lo, hi int, newW func(io.Writer, int) (io.WriteCloser, error)) coding {
	return coding{
		name: name, def: def, clamp: clampRange(lo, hi),
		encode: func(dst io.Writer, src io.Reader, level int) error {
			cw, err := newW(dst, level)
			if err != nil {
				return err
			}
			if _, err := io.Copy(cw, src); err != nil {
				cw.Close()
				return err
			}
			return cw.Close()
		},
	}
}

func oneShot(name string, def, lo, hi int, compress func(data []byte, level int) ([]byte, error)) coding {
	return coding{
		name: name, def: def, clamp: clampRange(lo, hi),
		encode: func(dst io.Writer, src io.Reader, level int) error {
			data, err := io.ReadAll(src)
			if err != nil {
				return err
			}
			out, err := compress(data, level)
			if err != nil {
				return err
			}
			_, err = dst.Write(out)
			return err
		},
	}
}

var gzipCoding = pooledStreaming("gzip", 6, 1, 9, func(dst io.Writer, level int) (resettable, error) {
	return gzip.NewWriterLevel(dst, level)
})

func parseAcceptEncoding(header string) (map[string]float64, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil, false
	}
	out := map[string]float64{}
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(part, ";")
		name := strings.ToLower(strings.TrimSpace(fields[0]))
		if name == "" {
			continue
		}
		q := 1.0
		for _, p := range fields[1:] {
			k, v, ok := strings.Cut(p, "=")
			if !ok {
				continue
			}
			if strings.ToLower(strings.TrimSpace(k)) == "q" {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					q = f
				}
			}
		}
		out[name] = q
	}
	return out, true
}

func negotiate(codings []coding, header string) (*coding, bool) {
	accepted, has := parseAcceptEncoding(header)
	wildcardQ, hasWildcard := accepted["*"]

	var best *coding
	var bestQ float64
	for i := range codings {
		c := &codings[i]
		q := 0.0
		switch {
		case !has:
			q = 1
		default:
			if v, ok := accepted[c.name]; ok {
				q = v
			} else if hasWildcard {
				q = wildcardQ
			}
		}
		if q <= 0 || (best != nil && q <= bestQ) {
			continue
		}
		best, bestQ = c, q
	}
	return best, best != nil
}

func resolveLevel(c *coding, raw string) int {
	if raw == "" {
		return c.def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return c.def
	}
	return c.clamp(n)
}

type server struct {
	file    string
	codings []coding
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(s.file)
	if err != nil {
		http.Error(w, "cannot open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Vary", "Accept-Encoding")
	if ct := mime.TypeByExtension(filepath.Ext(s.file)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	c, ok := negotiate(s.codings, r.Header.Get("Accept-Encoding"))
	if !ok {
		io.Copy(w, f)
		return
	}
	level := resolveLevel(c, r.URL.Query().Get("level"))

	w.Header().Set("Content-Encoding", c.name)
	c.encode(w, f, level)
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Fprintf(os.Stderr, "usage: %s <port> <file> [dictionary]\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}
	port, file := os.Args[1], os.Args[2]

	if _, err := os.Stat(file); err != nil {
		fmt.Fprintf(os.Stderr, "file %q: %v\n", file, err)
		os.Exit(1)
	}

	cs := baseCodings
	if len(os.Args) == 4 {
		dict, err := os.ReadFile(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "dictionary %q: %v\n", os.Args[3], err)
			os.Exit(1)
		}
		dc, err := dictCodings(dict)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dictionary %q: %v\n", os.Args[3], err)
			os.Exit(1)
		}
		// Dict codings go first so a client that offers them wins the tie.
		cs = append(dc, baseCodings...)
	}

	s := &server{file: file, codings: cs}
	http.HandleFunc("/", s.handle)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
