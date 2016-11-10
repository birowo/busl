package busltee

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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
	tmpfile, err := ioutil.TempFile("", "busltee_buffer")
	if err != nil {
		return nil, err
	}
	defer tmpfile.Close()
	t.body = &bodyReader{req.Body, tmpfile, nil}
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
			err = t.body.Reset()
			if err != nil {
				return nil, err
			}
			return t.tries(req)
		} else {
			return nil, err
		}
	} else {
		t.retries = 0
	}
	return res, err
}

type bodyReader struct {
	streamer   io.Reader
	buffWriter *os.File
	buffReader *os.File
}

func (*bodyReader) Close() error { return nil }
func (b *bodyReader) Reset() error {
	file, err := os.Open(b.buffWriter.Name())
	b.buffReader = file
	return err
}

func (b *bodyReader) Read(p []byte) (int, error) {
	if b.buffReader == nil {
		n, err := b.streamer.Read(p)
		if n > 0 {
			if n, err := b.buffWriter.Write(p[:n]); err != nil {
				return n, err
			}
		}
		return n, err
	} else {
		n, err := b.buffReader.Read(p)
		if err == io.EOF {
			b.buffReader = nil
			err = nil
		}

		return n, err
	}
}
