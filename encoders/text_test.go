package encoders

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type textTable struct {
	offset int64
	input  string
	output string
}

var (
	testTextData = []textTable{
		{0, "hello", "hello"},
		{0, "hello\n", "hello\n"},
		{0, "hello\nworld", "hello\nworld"},
		{0, "hello\nworld\n", "hello\nworld\n"},
		{1, "hello\nworld\n", "ello\nworld\n"},
		{6, "hello\nworld\n", "world\n"},
		{11, "hello\nworld\n", "\n"},
		{12, "hello\nworld\n", ""},
	}
)

func TestTextNoNewline(t *testing.T) {
	for _, data := range testTextData {
		r := &readSeekerCloser{strings.NewReader(data.input)}
		enc := NewTextEncoder(r)
		enc.Seek(data.offset, 0)
		assert.Equal(t, data.output, readstring(enc))
	}
}

func TestTextNonSeekableReader(t *testing.T) {
	// Seek the underlying reader before
	// passing to LimitReader: comparably similar
	// to scenario when reading from an http.Response
	r := &readSeekerCloser{strings.NewReader("hello world")}
	r.Seek(10, 0)

	// Use LimitReader to hide the Seeker interface
	lr := &limitedReadCloser{io.LimitedReader{r, 11}}

	enc := NewTextEncoder(lr)
	enc.Seek(10, 0)

	// `id` should be 11 even though the underlying
	// reader wasn't seeked at all.
	assert.Equal(t, "d", readstring(enc))
}
