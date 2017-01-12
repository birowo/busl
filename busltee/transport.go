package busltee

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
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

	log.Printf(
		"count#busltee.streamer.start=1 request_id=%s url=%s",
		req.Header.Get("Request-Id"),
		req.URL,
	)
	res, err := t.Transport.RoundTrip(newReq)
	var statusCode int
	if res != nil {
		statusCode = res.StatusCode
	}
	log.Printf(
		"count#busltee.streamer.end=1 request_id=%s url=%s err=%q status=%d",
		req.Header.Get("Request-Id"),
		req.URL,
		err,
		statusCode,
	)
	newReq.Body.Close()

	if err != nil || statusCode/100 != 2 {
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
	return &bodyReader{
		&sync.Mutex{},
		io.MultiReader(data, io.TeeReader(streamer, buffer)),
	}, nil
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
	return bytes.NewBuffer(d), err
}

type bodyReader struct {
	mutex  *sync.Mutex
	reader io.Reader
}

func (b *bodyReader) Close() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.reader = nil
	return nil
}

func (b *bodyReader) Read(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.reader == nil {
		return 0, io.EOF
	}

	return b.reader.Read(p)
}
