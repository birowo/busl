package encoders

import (
	"bytes"
	"fmt"
	"io"
)

const (
	id   = "id: %d\n"
	data = "data: %s\n"
)

type sseEncoder struct {
	reader io.Reader // stores the original reader
	offset int64     // offset for Seek purposes
}

// NewSSEEncoder creates a new server-sent event encoder
func NewSSEEncoder(r io.Reader) io.ReadSeeker {
	return &sseEncoder{reader: r}
}

func (r *sseEncoder) Seek(offset int64, whence int) (n int64, err error) {
	if seeker, ok := r.reader.(io.ReadSeeker); ok {
		r.offset, err = seeker.Seek(offset, whence)
	} else {
		// The underlying reader doesn't support seeking, but
		// we should still update the offset so the IDs will
		// properly reflect the adjusted offset.
		r.offset += offset
	}

	return r.offset, err
}

// FIXME: this version is simplified and assumes
// that len(p) is always greater than the potential
// length of data to be read.
func (r *sseEncoder) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)

	if n > 0 {
		buf := format(r.offset, p[:n])
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
