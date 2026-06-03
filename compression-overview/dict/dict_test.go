// usage
// CORPUS_DIR=<corpus dir> DICT_DIR=<dict dir> go test -run '^$' -bench '.' -benchtime 2s -count 6 -cpu 1
// corpus dir is the dir that contains the files you want to compress during benchmark
// dict dir should contain the tested dir suffixed with .bin extension
package dict

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/molecule-man/go-brrr"
)

var (
	brotliLevels = []int{3, 4, 5, 6, 7, 8, 9, 10, 11}
	zstdLevels   = []int{1, 3, 7, 11}
)

type compressFn func(src []byte) (int, error)

func loadCorpus(b *testing.B) (files [][]byte, total int64) {
	dir := os.Getenv("CORPUS_DIR")
	if dir == "" {
		b.Fatal("CORPUS_DIR is not set")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatalf("read corpus dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			b.Fatalf("read corpus file %s: %v", e.Name(), err)
		}
		files = append(files, data)
		total += int64(len(data))
	}
	if len(files) == 0 {
		b.Fatalf("no files in CORPUS_DIR=%s", dir)
	}
	return files, total
}

func listDicts(b *testing.B) []string {
	dir := os.Getenv("DICT_DIR")
	if dir == "" {
		b.Fatal("DICT_DIR is not set")
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.bin"))
	if err != nil {
		b.Fatalf("glob dicts: %v", err)
	}
	if len(paths) == 0 {
		b.Fatalf("no *.bin dicts in DICT_DIR=%s", dir)
	}
	sort.Strings(paths)
	return paths
}

func zstdCompressor(dict []byte, level int) func(*testing.B) compressFn {
	return func(b *testing.B) compressFn {
		opts := []zstd.EOption{
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
			zstd.WithEncoderConcurrency(1),
		}
		if dict != nil {
			opts = append(opts, zstd.WithEncoderDictRaw(0, dict))
		}
		enc, err := zstd.NewWriter(nil, opts...)
		if err != nil {
			b.Fatalf("zstd encoder: %v", err)
		}
		buf := make([]byte, 0, 1<<16)
		return func(src []byte) (int, error) {
			return len(enc.EncodeAll(src, buf[:0])), nil
		}
	}
}

func brotliCompressor(dict []byte, level int) func(*testing.B) compressFn {
	return func(b *testing.B) compressFn {
		var opts brrr.WriterOptions
		if dict != nil {
			pd, err := brrr.PrepareDictionary(dict)
			if err != nil {
				b.Fatalf("prepare brotli dict: %v", err)
			}
			opts.Dictionaries = []*brrr.PreparedDictionary{pd}
		}
		var buf bytes.Buffer
		w, err := brrr.NewWriterOptions(&buf, level, opts)
		if err != nil {
			b.Fatalf("brotli writer: %v", err)
		}
		return func(src []byte) (int, error) {
			buf.Reset()
			w.Reset(&buf)
			if _, err := w.Write(src); err != nil {
				return 0, err
			}
			if err := w.Close(); err != nil {
				return 0, err
			}
			return buf.Len(), nil
		}
	}
}

func run(build func(*testing.B) compressFn, corpus [][]byte, orig int64) func(*testing.B) {
	return func(b *testing.B) {
		comp := build(b)
		b.SetBytes(orig)
		b.ReportAllocs()

		var total int
		for b.Loop() {
			total = 0
			for _, f := range corpus {
				n, err := comp(f)
				if err != nil {
					b.Fatalf("compress: %v", err)
				}
				total += n
			}
		}
		b.ReportMetric(float64(total), "comp_bytes")
		b.ReportMetric(float64(orig)/float64(total), "ratio")
	}
}

func BenchmarkCompress(b *testing.B) {
	corpus, orig := loadCorpus(b)

	type dictEntry struct {
		name string
		data []byte // nil for the no-dictionary baseline
	}
	dicts := []dictEntry{{name: "nodict"}}
	for _, path := range listDicts(b) {
		data, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("read dict %s: %v", path, err)
		}
		dicts = append(dicts, dictEntry{name: filepath.Base(path), data: data})
	}

	for _, d := range dicts {
		for _, lvl := range zstdLevels {
			b.Run(fmt.Sprintf("dict=%s/lib=zstd/level=%d", d.name, lvl),
				run(zstdCompressor(d.data, lvl), corpus, orig))
		}
		for _, lvl := range brotliLevels {
			b.Run(fmt.Sprintf("dict=%s/lib=brotli/level=%d", d.name, lvl),
				run(brotliCompressor(d.data, lvl), corpus, orig))
		}
	}
}
