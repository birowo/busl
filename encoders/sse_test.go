package encoders

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type sseTable struct {
	offset int64
	input  string
	output string
}

var (
	testSSEData = []sseTable{
		{0, "hello", "id: 5\ndata: hello\n\n"},
		{0, "hello\n", "id: 6\ndata: hello\ndata: \n\n"},
		{0, "hello\nworld", "id: 11\ndata: hello\ndata: world\n\n"},
		{0, "hello\nworld\n", "id: 12\ndata: hello\ndata: world\ndata: \n\n"},
		{1, "hello\nworld\n", "id: 12\ndata: ello\ndata: world\ndata: \n\n"},
		{6, "hello\nworld\n", "id: 12\ndata: world\ndata: \n\n"},
		{11, "hello\nworld\n", "id: 12\ndata: \ndata: \n\n"},
		{12, "hello\nworld\n", ""},
	}
)

func TestSSENoNewline(t *testing.T) {
	for _, data := range testSSEData {
		r := strings.NewReader(data.input)
		enc := NewSSEEncoder(r)
		enc.Seek(data.offset, 0)
		assert.Equal(t, data.output, readstring(enc))
	}
}

func TestSSENonSeekableReader(t *testing.T) {
	// Seek the underlying reader before
	// passing to LimitReader: comparably similar
	// to scenario when reading from an http.Response
	r := strings.NewReader("hello world")
	r.Seek(10, 0)

	// Use LimitReader to hide the Seeker interface
	lr := io.LimitReader(r, 11)

	enc := NewSSEEncoder(lr)
	enc.Seek(10, 0)

	// `id` should be 11 even though the underlying
	// reader wasn't seeked at all.
	assert.Equal(t, "id: 11\ndata: d\n\n", readstring(enc))
}

func readstring(r io.Reader) string {
	buf, _ := ioutil.ReadAll(r)
	return string(buf)
}
