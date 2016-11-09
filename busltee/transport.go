package busltee

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"
)

var ErrTooManyRetries = errors.New("Reached max retries")

type Transport struct {
	retries       uint
	MaxRetries    uint
	Transport     http.RoundTripper
	SleepDuration time.Duration
	body          *bodyReader
}

func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	t.body = &bodyReader{req.Body, &bytes.Buffer{}, true}
	if t.Transport == nil {
		t.Transport = &http.Transport{}
	}
	if t.SleepDuration == 0 {
		t.SleepDuration = time.Second
	}

	req.Body = t.body
	req.ContentLength = -1
	return t.tries(req)
}

func (t *Transport) tries(req *http.Request) (*http.Response, error) {
	res, err := t.Transport.RoundTrip(req)

	if err != nil || res.StatusCode/100 != 2 {
		if t.retries < t.MaxRetries {
			time.Sleep(t.SleepDuration)
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
