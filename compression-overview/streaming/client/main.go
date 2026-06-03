package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"blog-examples/compression-overview/streaming/proto"

	"github.com/molecule-man/go-brrr"
)

func main() {
	logPath := flag.String("log", "", "append per-frame compressed sizes to this CSV file")
	dictPath := flag.String("dict", "", "shared dictionary file (required for dict mode)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-dict file] [-log file] <url>\n", progName())
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0), *dictPath, *logPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url, dictPath, logPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	mode := resp.Header.Get(proto.ModeHeader)
	if !proto.ValidMode(mode) {
		return fmt.Errorf("server reported unknown mode %q", mode)
	}

	var dict []byte
	if mode == proto.ModeDict {
		if dictPath == "" {
			return fmt.Errorf("mode %q requires -dict", mode)
		}
		if dict, err = os.ReadFile(dictPath); err != nil {
			return err
		}
	}

	var logw *os.File
	if logPath != "" {
		if logw, err = os.Create(logPath); err != nil {
			return err
		}
		defer logw.Close()
		fmt.Fprintln(logw, "mode,frame,compressed_bytes")
	}

	fmt.Printf("connected: mode=%s\n", mode)

	// In continuous mode every frame is a slice of one brotli stream, so we feed
	// the decoded bytes into a single Reader and pull length-prefixed events off
	// it in a goroutine. Frame boundaries and event boundaries do not align.
	var pw *io.PipeWriter
	decoded := make(chan struct{})
	if !proto.Independent(mode) {
		var pr *io.PipeReader
		pr, pw = io.Pipe()
		go func() {
			defer close(decoded)
			drainStream(brrr.NewReader(pr))
		}()
	} else {
		close(decoded)
	}

	total, serverTotal, err := consume(resp.Body, mode, dict, logw, pw)
	if pw != nil {
		pw.Close() // signal EOF to the decoder goroutine
	}
	<-decoded
	if err != nil {
		return err
	}

	fmt.Printf("\ntransferred %d compressed bytes across the stream\n", total)
	if serverTotal >= 0 && serverTotal != total {
		fmt.Printf("WARNING: server reported %d, client counted %d\n", serverTotal, total)
	}
	return nil
}

func consume(body io.Reader, mode string, dict []byte, logw io.Writer, pw io.Writer) (total, serverTotal int, err error) {
	serverTotal = -1
	r := bufio.NewReader(body)

	var (
		event   string
		data    strings.Builder
		id      string
		hasData bool
	)
	dispatch := func() error {
		switch {
		case event == "done":
			serverTotal, _ = strconv.Atoi(data.String())
		case hasData:
			n, err := handleFrame(mode, dict, id, data.String(), logw, pw)
			if err != nil {
				return err
			}
			total += n
		}
		event, id, hasData = "", "", false
		data.Reset()
		return nil
	}

	for {
		line, readErr := r.ReadString('\n')
		if len(line) > 0 {
			switch line = strings.TrimRight(line, "\r\n"); {
			case line == "":
				if err := dispatch(); err != nil {
					return total, serverTotal, err
				}
			case strings.HasPrefix(line, ":"): // comment
			case strings.HasPrefix(line, "id:"):
				id = strings.TrimSpace(line[len("id:"):])
			case strings.HasPrefix(line, "event:"):
				event = strings.TrimSpace(line[len("event:"):])
			case strings.HasPrefix(line, "data:"):
				hasData = true
				data.WriteString(strings.TrimPrefix(line[len("data:"):], " "))
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return total, serverTotal, readErr
			}
			return total, serverTotal, nil
		}
	}
}

func handleFrame(mode string, dict []byte, id, b64 string, logw io.Writer, pw io.Writer) (int, error) {
	chunk, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, fmt.Errorf("frame %s: bad base64: %w", id, err)
	}
	if logw != nil {
		fmt.Fprintf(logw, "%s,%s,%d\n", mode, id, len(chunk))
	}

	if proto.Independent(mode) {
		payload, err := decodeFrame(chunk, dict, mode)
		if err != nil {
			return len(chunk), fmt.Errorf("frame %s: %w", id, err)
		}
		fmt.Printf("frame %-3s %6d B  %s\n", id, len(chunk), preview(payload))
		return len(chunk), nil
	}

	fmt.Printf("frame %-3s %6d B  (continuous stream)\n", id, len(chunk))
	if len(chunk) > 0 {
		if _, err := pw.Write(chunk); err != nil {
			return len(chunk), err
		}
	}
	return len(chunk), nil
}

func decodeFrame(chunk, dict []byte, mode string) ([]byte, error) {
	var rd *brrr.Reader
	if mode == proto.ModeDict {
		var err error
		if rd, err = brrr.NewReaderOptions(bytes.NewReader(chunk), brrr.ReaderOptions{
			Dictionaries: [][]byte{dict},
		}); err != nil {
			return nil, err
		}
	} else {
		rd = brrr.NewReader(bytes.NewReader(chunk))
	}
	plain, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	return proto.ReadRecord(bytes.NewReader(plain))
}

func drainStream(rd io.Reader) {
	for n := 0; ; n++ {
		rec, err := proto.ReadRecord(rd)
		if err != nil {
			if err != io.EOF && err != io.ErrClosedPipe && err != io.ErrUnexpectedEOF {
				fmt.Fprintf(os.Stderr, "decode: %v\n", err)
			}
			return
		}
		fmt.Printf("  └ event %-3d %s\n", n, preview(rec))
	}
}

func preview(b []byte) string {
	r := []rune(strings.Join(strings.Fields(string(b)), " "))
	if len(r) <= 33 {
		return string(r)
	}
	return string(r[:20]) + "..." + string(r[len(r)-10:])
}

func progName() string {
	if len(os.Args) > 0 {
		if i := strings.LastIndexByte(os.Args[0], '/'); i >= 0 {
			return os.Args[0][i+1:]
		}
		return os.Args[0]
	}
	return "client"
}
