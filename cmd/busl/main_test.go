package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStorageBaseURL(t *testing.T) {
	os.Setenv("EXAMPLE_COM_STORAGE_BASE_URL", "https://example.s3.amazonaws.com/")
	os.Setenv("STORAGE_BASE_URL", "default")

	assert.Equal(t, "https://example.s3.amazonaws.com/",
		getStorageBaseURL(&http.Request{Host: "example.com"}))
	assert.Equal(t, "default",
		getStorageBaseURL(&http.Request{Host: "localhost"}))
	assert.Equal(t, "default",
		getStorageBaseURL(&http.Request{}))
}
