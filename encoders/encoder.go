package encoders

import "io"

type Encoder interface {
	io.Reader
	io.Closer
	io.Seeker
}
