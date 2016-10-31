package busltee

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

var ErrTooManyRetries = errors.New("Reached max retries")

type Transport struct {
	retries    uint
	MaxRetries uint
	Transport  http.RoundTripper
	body       *bodyReader
}

func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	t.body = &bodyReader{req.Body, &bytes.Buffer{}, true}
	req.Body = t.body
	req.ContentLength = -1
	return t.tries(req)
}

func (t *Transport) tries(req *http.Request) (*http.Response, error) {
	res, err := t.Transport.RoundTrip(req)

	if err != nil {
		if t.retries < t.MaxRetries {
			t.retries += 1
			t.body.Reset()
			return t.tries(req)
		} else {
			return nil, err
		}
	}

	return res, err
}

type bodyReader struct {
	r          io.Reader
	w          *bytes.Buffer
	bufferSent bool
}

func (*bodyReader) Close() error { return nil }
func (b *bodyReader) Reset() {
	b.bufferSent = false
}

func (b *bodyReader) Read(p []byte) (int, error) {
	if b.bufferSent {
		n, err := b.r.Read(p)
		if n > 0 {
			if n, err := b.w.Write(p[:n]); err != nil {
				return n, err
			}
		}
		return n, err
	} else {
		b.bufferSent = true
		n, err := bytes.NewBuffer(b.w.Bytes()).Read(p)
		if err == io.EOF {
			err = nil
		}

		return n, err
	}
}
