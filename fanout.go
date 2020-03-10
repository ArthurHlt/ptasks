package ptasks

import (
	"io"
)

type FanoutReader struct {
	source  io.Reader
	writers MultiWriters
}

type MultiWriters []io.Writer

func (m MultiWriters) Write(p []byte) (n int, err error) {
	for _, w := range m {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func NewFanoutReader(source io.Reader) *FanoutReader {
	return &FanoutReader{
		source:  source,
		writers: make([]io.Writer, 0),
	}
}

func (f *FanoutReader) AddWriter(writer io.Writer) {
	f.writers = append(f.writers, writer)
}

func (f *FanoutReader) Fanout() error {
	_, err := io.Copy(f.writers, f.source)
	if rc, ok := f.source.(io.ReadCloser); ok {
		rc.Close()
	}
	for _, w := range f.writers {
		if wc, ok := w.(io.WriteCloser); ok {
			wc.Close()
		}
	}
	return err
}
