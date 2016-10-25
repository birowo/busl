package encoders

import (
	"errors"
	"io"
)

type textEncoder struct {
	io.Reader       // stores the original reader
	offset    int64 // offset for Seek purposes
}

// NewTextEncoder creates a text events encoder
func NewTextEncoder(r io.Reader) io.ReadSeeker {
	return &textEncoder{Reader: r}
}

func (r *textEncoder) Seek(offset int64, whence int) (n int64, err error) {
	if seeker, ok := r.Reader.(io.ReadSeeker); ok {
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
