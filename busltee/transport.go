package busltee

import (
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

	bufferName string
	mutex      *sync.Mutex
	closed     bool
}

func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	tmpFile, err := ioutil.TempFile("", "busltee_buffer")
	if err != nil {
		return nil, err
	}
	t.bufferName = tmpFile.Name()
	t.mutex = &sync.Mutex{}

	go func() {
		defer tmpFile.Close()
		defer t.Close()

		tee := io.TeeReader(req.Body, tmpFile)
		_, err := ioutil.ReadAll(tee)
		if err != nil {
			log.Fatal(err)
		}
	}()

	if t.Transport == nil {
		t.Transport = &http.Transport{}
	}
	if t.SleepDuration == 0 {
		t.SleepDuration = time.Second
	}
	return t.tries(req)
}

func (t *Transport) tries(req *http.Request) (*http.Response, error) {
	res, err := t.runRequest(req)

	if err != nil || res.StatusCode/100 != 2 {
		if t.retries < t.MaxRetries {
			time.Sleep(t.SleepDuration)
			t.retries += 1
			return t.tries(req)
		}
	} else {
		t.retries = 0
	}
	return res, err
}

func (t *Transport) runRequest(req *http.Request) (*http.Response, error) {
	var statusCode int
	bodyReader, err := t.newBodyReader()
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
	newReq.Body.Close()
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
	return res, err
}

func (t *Transport) newBodyReader() (io.ReadCloser, error) {
	reader, err := os.Open(t.bufferName)
	if err != nil {
		return nil, err
	}
	return &bodyReader{reader, t}, nil
}

func (t *Transport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.closed = true
	return nil
}

func (t *Transport) isClosed() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.closed
}

type bodyReader struct {
	io.ReadCloser
	t *Transport
}

func (b *bodyReader) Read(p []byte) (int, error) {
	for {
		n, err := b.ReadCloser.Read(p)
		if err == io.EOF && !b.t.isClosed() {
			err = nil
		}

		if n > 0 || err != nil {
			return n, err
		}
	}
}
