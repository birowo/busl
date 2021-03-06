package encoders

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	id   = "id: %d\n"
	data = "data: %s\n"
)

type sseEncoder struct {
	io.ReadCloser       // stores the original reader
	offset        int64 // offset for Seek purposes
}

// NewSSEEncoder creates a new server-sent event encoder
func NewSSEEncoder(r io.ReadCloser) Encoder {
	return &sseEncoder{ReadCloser: r}
}

func (r *sseEncoder) Seek(offset int64, whence int) (n int64, err error) {
	if seeker, ok := r.ReadCloser.(io.ReadSeeker); ok {
		r.offset, err = seeker.Seek(offset, whence)
	} else {
		// The underlying reader doesn't support seeking, but
		// we should still update the offset so the IDs will
		// properly reflect the adjusted offset.

		if whence != io.SeekStart {
			return 0, errors.New("Only SeekStart is supported")
		}
		r.offset += offset
	}

	return r.offset, err
}

func (r *sseEncoder) Read(p []byte) (n int, err error) {
	// We assume SSE won't add more than twice the amount of data we get
	q := make([]byte, len(p)/2)
	n, err = r.ReadCloser.Read(q)

	if n > 0 {
		buf := format(r.offset, q[:n])
		if len(buf) > len(p) {
			return 0, errors.New("buffer length cannot be higher than bytes array")
		}

		r.offset += int64(n)
		n = copy(p, buf)
	}

	return n, err
}

func format(pos int64, msg []byte) []byte {
	buf := bytes.NewBufferString(fmt.Sprintf(id, pos+int64(len(msg))))

	for _, line := range bytes.Split(msg, []byte{'\n'}) {
		buf.WriteString(fmt.Sprintf(data, line))
	}
	buf.Write([]byte{'\n'})

	return buf.Bytes()
}
