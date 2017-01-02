package busltee

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

var ErrTooManyRetries = errors.New("Reached max retries")

type Transport struct {
	retries       uint
	MaxRetries    uint
	Transport     http.RoundTripper
	SleepDuration time.Duration

	body   io.ReadCloser
	buffer *os.File
}

func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	tmpFile, err := ioutil.TempFile("", "busltee_buffer")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()
	t.buffer = tmpFile
	t.body = req.Body

	if t.Transport == nil {
		t.Transport = &http.Transport{}
	}
	if t.SleepDuration == 0 {
		t.SleepDuration = time.Second
	}
	return t.tries(req)
}

func (t *Transport) tries(req *http.Request) (*http.Response, error) {
	bodyReader, err := newBodyReader(t.body, t.buffer)
	if err != nil {
		return nil, err
	}
	newReq, err := http.NewRequest(req.Method, req.URL.String(), bodyReader)
	newReq.Header = req.Header
	res, err := t.Transport.RoundTrip(newReq)
	newReq.Body.Close()

	if err != nil || res.StatusCode/100 != 2 {
		if t.retries < t.MaxRetries {
			time.Sleep(t.SleepDuration)
			t.retries += 1
			return t.tries(newReq)
		}
	} else {
		t.retries = 0
	}
	return res, err
}

func newBodyReader(streamer io.Reader, buffer *os.File) (*bodyReader, error) {
	data, err := readBuffer(buffer)
	if err != nil {
		return nil, err
	}
	return &bodyReader{streamer, buffer, data, &sync.Mutex{}}, nil
}

func readBuffer(b *os.File) (*bytes.Buffer, error) {
	f, err := os.Open(b.Name())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, nil
	}
	return bytes.NewBuffer(d), err
}

type bodyReader struct {
	streamer   io.Reader
	buffWriter *os.File
	buffReader *bytes.Buffer
	mutex      *sync.Mutex
}

func (b *bodyReader) Close() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.streamer = nil
	return nil
}
func (b *bodyReader) Read(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.streamer == nil {
		return 0, io.EOF
	}

	if b.buffReader == nil {
		n, err := b.streamer.Read(p)
		if err != nil {
			return n, err
		}
		if rn, err := b.buffWriter.Write(p[:n]); err != nil {
			return rn, err
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
