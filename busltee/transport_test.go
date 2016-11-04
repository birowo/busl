package busltee

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoError(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(body[:len(body)]) != "hello world" {
			t.Fatalf("Expected body to be 'hello world'. Got '%q'", body)
		}
	})
	client := &http.Client{Transport: &Transport{}}
	res, err := client.Post(server.URL, "", bytes.NewBuffer([]byte("hello world")))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("was expecting 200 got %d", res.StatusCode)
	}
}

func TestDisconnection(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	var callCount int
	var expectedBody string
	var stdin bytes.Buffer
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		stdin.Write([]byte("hello world\n"))

		if string(body[:len(body)]) != expectedBody {
			t.Fatalf("Unexpected body. Expected %q. Got %q", expectedBody, body)
		}
		expectedBody += "hello world\n"
		callCount += 1

		if callCount < 4 {
			server.CloseClientConnections()
		}
	})
	transport := &Transport{
		MaxRetries: 5,
	}
	client := &http.Client{Transport: transport}
	res, err := client.Post(server.URL, "", &stdin)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("was expecting 200 got %d", res.StatusCode)
	}

	if callCount != 4 {
		t.Fatalf("was expecting 5 retries. Got %d", callCount)
	}
}

func TestHTTPError(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	var callCount int
	var expectedBody string
	var stdin bytes.Buffer
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		stdin.Write([]byte("hello world\n"))

		if string(body[:len(body)]) != expectedBody {
			t.Fatalf("Unexpected body. Expected %q. Got %q", expectedBody, body)
		}
		expectedBody += "hello world\n"
		callCount += 1

		if callCount < 4 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	transport := &Transport{
		MaxRetries: 5,
	}
	client := &http.Client{Transport: transport}
	res, err := client.Post(server.URL, "", &stdin)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("was expecting 200 got %d", res.StatusCode)
	}

	if callCount != 4 {
		t.Fatalf("was expecting 5 retries. Got %d", callCount)
	}
}
