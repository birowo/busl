package safebuffer

import (
	"bytes"
	"io"
	"sync"
)

func NewMock() io.ReadWriter {
	return &mockStdin{
		&bytes.Buffer{},
		&sync.Mutex{},
	}
}

type mockStdin struct {
	b *bytes.Buffer
	m *sync.Mutex
}

func (m *mockStdin) Read(p []byte) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.b.Read(p)
}

func (m *mockStdin) Write(p []byte) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.b.Write(p)
}
