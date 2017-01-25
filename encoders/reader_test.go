package encoders

import "io"

type readSeekerCloser struct {
	io.ReadSeeker
}

func (r *readSeekerCloser) Close() error {
	return nil
}

type limitedReadCloser struct {
	*io.LimitedReader
}

func (r *limitedReadCloser) Close() error {
	return nil
}
