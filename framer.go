package easytcp

import (
	"bytes"
	"io"
)

// Framer customizes how frames are received from and sent to the connection.
//
// A frame is the byte payload handed to Packer.Unpack on receive,
// and the byte payload returned by Packer.Pack on send.
// Typical uses: length-prefixed envelopes, delimiters, compression, encryption.
//
// If Framer is nil, the Packer reads from and the session writes to the connection directly.
type Framer interface {
	// ReadFrame reads one complete frame from r.
	// Returning io.EOF ends the session quietly; any other error ends it with an error log.
	ReadFrame(r io.Reader) ([]byte, error)

	// WriteFrame writes frame to w as one complete frame.
	WriteFrame(w io.Writer, frame []byte) error
}

// unpackFrame reads a message from r through f (if any) and p.
func unpackFrame(f Framer, p Packer, r io.Reader) (*Message, error) {
	if f == nil {
		return p.Unpack(r)
	}
	frame, err := f.ReadFrame(r)
	if err != nil {
		return nil, err
	}
	return p.Unpack(bytes.NewReader(frame))
}

// writeFrame writes data to w through f (if any).
func writeFrame(f Framer, w io.Writer, data []byte) error {
	if f == nil {
		_, err := w.Write(data)
		return err
	}
	return f.WriteFrame(w, data)
}
