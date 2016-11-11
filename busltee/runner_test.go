package busltee

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var conf = &Config{
	Timeout:       1,
	SleepDuration: time.Millisecond,
}

func TestStreamNoURL(t *testing.T) {
	err := streamNoRetry("", strings.NewReader(""), conf)

	if err != errMissingURL {
		t.Fatalf("Expected err to be %v", errMissingURL)
	}
}

func TestStreamTimeout(t *testing.T) {
	err := streamNoRetry("http://10.255.255.1", strings.NewReader(""), conf)

	if !isTimeout(err) {
		t.Fatalf("Expected err to be a timeout error, got %v", err)
	}
}

func TestStreamConnRefused(t *testing.T) {
	err := streamNoRetry("http://0.0.0.0:0", strings.NewReader(""), conf)

	if err == nil {
		t.Fatalf("Expected err to be non-nil, got %v", err)
	}
}

func TestStreamDoesNotCloseReader(t *testing.T) {
	r, w := io.Pipe()
	streamNoRetry("http://0.0.0.0:0", r, conf)
	go func() {
		p := make([]byte, 10)
		r.Read(p)
	}()

	_, err := w.Write([]byte{0})
	if err != nil {
		t.Fatalf("Expected Write to work, got error %v", err)
	}
}

func TestStreamPost(t *testing.T) {
	server, post := fauxBusl()
	defer server.Close()

	r, w := io.Pipe()
	done := make(chan struct{})

	go func() {
		streamNoRetry(server.URL, r, conf)
		close(done)
	}()

	w.Write([]byte("hello world"))
	w.Close()

	<-done

	select {
	case result := <-post:
		if string(result) != "hello world" {
			t.Fatalf("Expected POST body to be `hello world`, got %s", result)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("POST channel got no response")
	}
}

func Test_run(t *testing.T) {
	r, w := io.Pipe()

	buf := &bytes.Buffer{}

	go func() {
		io.Copy(buf, r)
	}()
	run([]string{"printf", "hello"}, w, w)

	if out := buf.Bytes(); string(out) != "hello" {
		t.Fatalf("Expected reader to have generated `hello`, got %s", out)
	}
}

func TestRun(t *testing.T) {
	server, post := fauxBusl()
	defer server.Close()

	if code := Run(server.URL, []string{"printf", "hello"}, &Config{}); code != 0 {
		t.Fatalf("Expected exit code to be 0, got %d", code)
	}

	select {
	case result := <-post:
		if string(result) != "hello" {
			t.Fatalf("Expected POST body to be `hello`, got %s", result)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("POST channel got no response")
	}
}

func TestRequestID(t *testing.T) {
	post := make(chan string, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		post <- r.Header.Get("Request-ID")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	config := &Config{
		RequestID: "2469a9df-5a5a-4f16-af0f-ee75aa252d50",
	}
	if code := Run(server.URL, []string{"printf", "hello"}, config); code != 0 {
		t.Fatalf("Expected exit code to be 0, got %d", code)
	}

	select {
	case result := <-post:
		if string(result) != config.RequestID {
			t.Fatalf("Expected POST body to be `%s`, got %s", config.RequestID, result)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("POST channel got no response")
	}

}

func fauxBusl() (*httptest.Server, chan []byte) {
	post := make(chan []byte, 10)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("Hit here")
		if r.Method == "POST" {
			println("POST got")
			b, _ := ioutil.ReadAll(r.Body)
			post <- b
		}
	})

	server := httptest.NewServer(mux)
	return server, post
}
