package easytcp

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// lenFramer prefixes each frame with a 2-byte big-endian length.
type lenFramer struct{}

func (lenFramer) ReadFrame(r io.Reader) ([]byte, error) {
	var h [2]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	b := make([]byte, binary.BigEndian.Uint16(h[:]))
	_, err := io.ReadFull(r, b)
	return b, err
}

func (lenFramer) WriteFrame(w io.Writer, f []byte) error {
	buf := make([]byte, 2, 2+len(f))
	binary.BigEndian.PutUint16(buf, uint16(len(f)))
	_, err := w.Write(append(buf, f...))
	return err
}

func TestFramer_roundTrip(t *testing.T) {
	p := NewDefaultPacker()
	data, err := p.Pack(NewMessage(7, []byte("hi")))
	assert.NoError(t, err)

	var wire bytes.Buffer
	assert.NoError(t, writeFrame(lenFramer{}, &wire, data))
	assert.Equal(t, append([]byte{0, byte(len(data))}, data...), wire.Bytes())

	msg, err := unpackFrame(lenFramer{}, p, &wire)
	assert.NoError(t, err)
	assert.Equal(t, 7, msg.ID())
	assert.Equal(t, []byte("hi"), msg.Data())
}

func TestFramer_nilIsDirect(t *testing.T) {
	p := NewDefaultPacker()
	var wire bytes.Buffer
	data, _ := p.Pack(NewMessage(1, []byte("x")))
	assert.NoError(t, writeFrame(nil, &wire, data))
	assert.Equal(t, data, wire.Bytes())
	msg, err := unpackFrame(nil, p, &wire)
	assert.NoError(t, err)
	assert.Equal(t, []byte("x"), msg.Data())
}
