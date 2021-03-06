package stream

import (
	"bytes"

	containers "github.com/ernoaapa/eliot/pkg/api/services/containers/v1"
)

// Reader is io.Reader implementation what reads bytes from RPC stream
type Reader struct {
	buffer bytes.Buffer
	stream StdinStreamServer
}

// StdinStreamServer interface for the endpoint what takes stdin stream in
type StdinStreamServer interface {
	Recv() (*containers.StdinStreamRequest, error)
}

// StdinStreamClient interface for the client what sends stdin stream messages
type StdinStreamClient interface {
	Send(*containers.StdinStreamRequest) error
}

// NewReader creates new Reader instance
func NewReader(stream StdinStreamServer) *Reader {
	return &Reader{stream: stream}
}

// Write writes bytes to given RPC stream
func (w *Reader) Read(p []byte) (n int, err error) {
	if w.buffer.Len() == 0 {
		req, err := w.stream.Recv()
		if err != nil {
			return 0, err
		}
		w.buffer.Write(req.GetInput())
	}
	return w.buffer.Read(p)
}
