package storage

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/heroku/busl/util"
)

// Number of times we should retry a failed HTTP request.
const retries = 3

// storage errors
var (
	ErrNoStorage = errors.New("No storage defined")
	ErrNotFound  = errors.New("HTTP 404")
	ErrRange     = errors.New("HTTP 416: Invalid Range")
	Err5xx       = errors.New("HTTP 5xx")
)

// Put stores the given reader onto the underlying blob storage
// with the given requestURI. The requestURI is resolved
// using the `STORAGE_BASE_URL` as the base.
//
// Retries transient errors `retries` number of times.
//
// Usage:
//
//   reader := strings.NewReader("hello")
//   requestURI := "1/2/3?X-Amz-Algorithm=...&..."
//   err := storage.Put(requestURI, reader)
//
func Put(requestURI, baseURI string, reader io.Reader) (err error) {
	for i := retries; i > 0; i-- {
		err = put(requestURI, baseURI, reader)

		// Break if we get nil / any error other than Err5xx
		if err == nil {
			util.Count("storage.put.success")
			return nil
		}

		if err != Err5xx {
			util.Count("storage.put.error")
			return err
		}

		// Log the put retry
		util.Count("storage.put.retry")
	}

	// We've ran out of retries
	util.Count("storage.put.maxretries")
	return err
}

func put(requestURI, baseURI string, reader io.Reader) error {
	req, err := newRequest("PUT", requestURI, baseURI, reader)
	if err != nil {
		return err
	}
	res, err := process(req)
	if res != nil {
		defer res.Body.Close()
	}
	return err
}

// Get grabs the data stored in requestURI.
// The requestURI is resolved using the `STORAGE_BASE_URL` as the base.
//
// Retries transient errors `retries` number of times.
//
// Usage:
//
//   requestURI := "1/2/3?X-Amz-Algorithm=...&..."
//   reader, err := storage.Get(requestURI, 0)
//
func Get(requestURI, baseURI string, offset int64) (rd io.ReadCloser, err error) {
	for i := retries; i > 0; i-- {
		rd, err = get(requestURI, baseURI, offset)

		if err == nil {
			util.Count("storage.get.success")
			return rd, nil
		}

		if err != Err5xx {
			util.Count("storage.get.error")
			return rd, err
		}

		// Close the body immediately to prevent
		// file descriptor leaks.
		if rd != nil {
			rd.Close()
		}

		util.Count("storage.get.retry")
	}

	// We've ran out of retries
	util.Count("storage.get.maxretries")
	return rd, err
}

func get(requestURI, baseURI string, offset int64) (io.ReadCloser, error) {
	req, err := newRequest("GET", requestURI, baseURI, nil)
	if err != nil {
		return nil, err
	}
	req.TransferEncoding = []string{"chunked"}
	req.Header.Add("Transfer-Encoding", "chunked")

	if offset > 0 {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	res, err := process(req)
	if res == nil {
		return nil, err
	}
	return res.Body, err
}

// constructs an http.Request object, resolving requestURI
// under `STORAGE_BASE_URL`.
func newRequest(method, requestURI, baseURI string, reader io.Reader) (*http.Request, error) {
	u, err := absoluteURL(baseURI, requestURI)
	if err != nil {
		return nil, err
	}

	return http.NewRequest(method, u.String(), reader)
}

// Executes the HTTP request:
// Errors:
//
//   - Err5xx
//   - Err4xx
//   - ErrRange
//
func process(req *http.Request) (*http.Response, error) {
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	res, err := client.Do(req)
	if err == nil {
		switch {
		case res.StatusCode == 416:
			err = ErrRange
		case res.StatusCode == 404 || res.StatusCode == 403:
			err = ErrNotFound
		case res.StatusCode >= 500:
			err = Err5xx
		case res.StatusCode/100 != 2:
			err = fmt.Errorf("Expected 2xx, got %d", res.StatusCode)
		}
	}
	return res, err
}

func absoluteURL(baseURI, requestURI string) (*url.URL, error) {
	if ref, err := url.ParseRequestURI(normalize(requestURI)); err != nil {
		return nil, err
	} else if uri, err := baseURL(baseURI); err != nil {
		return nil, err
	} else {
		return uri.ResolveReference(ref), nil
	}
}
func baseURL(baseString string) (*url.URL, error) {
	if baseString == "" {
		return nil, ErrNoStorage
	}
	return url.Parse(baseString)
}

// Keep concept of requestURI similar to S3 without a slash prefix.
// We add the prefix to make `ParseRequestURI` happy.
func normalize(requestURI string) string {
	return "/" + requestURI
}
