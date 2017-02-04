package shadowaead

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"net"
)

// payloadSizeMask is the maximum size of payload in bytes.
const payloadSizeMask = 0x3FFF // 16*1024 - 1

type writer struct {
	io.Writer
	cipher.AEAD
	nonce []byte
	buf   []byte
}

// NewWriter wraps an io.Writer with AEAD encryption.
func NewWriter(w io.Writer, aead cipher.AEAD) io.Writer {
	return &writer{Writer: w, AEAD: aead}
}

func (w *writer) init() error {
	w.buf = make([]byte, 2+w.Overhead()+payloadSizeMask+w.Overhead())
	w.nonce = make([]byte, w.NonceSize())
	_, err := io.ReadFull(rand.Reader, w.nonce)
	if err != nil {
		return err
	}
	_, err = w.Writer.Write(w.nonce)
	return err
}

// Write encrypts b and writes to the embedded io.Writer.
func (w *writer) Write(b []byte) (int, error) {
	n, err := w.ReadFrom(bytes.NewBuffer(b))
	return int(n), err
}

// ReadFrom reads from the given io.Reader until EOF or error, encrypts and
// writes to the embedded io.Writer. Returns number of bytes read from r and
// any error encountered.
func (w *writer) ReadFrom(r io.Reader) (n int64, err error) {
	if w.nonce == nil {
		if err := w.init(); err != nil {
			return 0, err
		}
	}

	for {
		buf := w.buf
		payloadBuf := buf[2+w.Overhead() : 2+w.Overhead()+payloadSizeMask]
		nr, er := r.Read(payloadBuf)

		if nr > 0 {
			n += int64(nr)
			buf = buf[:2+w.Overhead()+nr+w.Overhead()]
			payloadBuf = payloadBuf[:nr]
			buf[0], buf[1] = byte(nr>>8), byte(nr) // big-endian payload size
			w.Seal(buf[:0], w.nonce, buf[:2], nil)
			increment(w.nonce)

			w.Seal(payloadBuf[:0], w.nonce, payloadBuf, nil)
			increment(w.nonce)

			_, ew := w.Writer.Write(buf)
			if ew != nil {
				err = ew
				break
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.ReaderFrom contract
				err = er
			}
			break
		}
	}

	return n, err
}

type reader struct {
	io.Reader
	cipher.AEAD
	nonce    []byte
	buf      []byte
	leftover []byte
}

// NewReader wraps an io.Reader with AEAD decryption.
func NewReader(r io.Reader, aead cipher.AEAD) io.Reader {
	return &reader{Reader: r, AEAD: aead}
}

func (r *reader) init() error {
	r.buf = make([]byte, payloadSizeMask+r.Overhead())
	r.nonce = make([]byte, r.NonceSize())
	_, err := io.ReadFull(r.Reader, r.nonce)
	return err
}

// read and decrypt a record into the internal buffer. Return decrypted payload length and any error encountered.
func (r *reader) read() (int, error) {
	if r.nonce == nil {
		if err := r.init(); err != nil {
			return 0, err
		}
	}

	// decrypt payload size
	buf := r.buf[:2+r.Overhead()]
	_, err := io.ReadFull(r.Reader, buf)
	if err != nil {
		return 0, err
	}

	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	increment(r.nonce)
	if err != nil {
		return 0, err
	}

	size := (int(buf[0])<<8 + int(buf[1])) & payloadSizeMask

	// decrypt payload
	buf = r.buf[:size+r.Overhead()]
	_, err = io.ReadFull(r.Reader, buf)
	if err != nil {
		return 0, err
	}

	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	increment(r.nonce)
	if err != nil {
		return 0, err
	}

	return size, nil
}

// Read reads from the embedded io.Reader, decrypts and writes to b.
func (r *reader) Read(b []byte) (int, error) {
	// copy decrypted bytes (if any) from previous record first
	if len(r.leftover) > 0 {
		n := copy(b, r.leftover)
		r.leftover = r.leftover[n:]
		return n, nil
	}

	n, err := r.read()
	m := copy(b, r.buf[:n])
	if m < n { // insufficient len(b), keep leftover for next read
		r.leftover = r.buf[m:n]
	}
	return m, err
}

// WriteTo reads from the embedded io.Reader, decrypts and writes to w until
// there's no more data to write or when an error occurs. Return number of
// bytes written to w and any error encountered.
func (r *reader) WriteTo(w io.Writer) (n int64, err error) {
	for {
		nr, er := r.read()
		if nr > 0 {
			nw, ew := w.Write(r.buf[:nr])
			n += int64(nw)

			if ew != nil {
				err = ew
				break
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.Copy contract (using src.WriteTo shortcut)
				err = er
			}
			break
		}
	}

	return n, err
}

type streamConn struct {
	net.Conn
	r *reader
	w *writer
}

type closeWriter interface {
	CloseWrite() error
}

type closeReader interface {
	CloseRead() error
}

func (c *streamConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *streamConn) WriteTo(w io.Writer) (int64, error) {
	return c.r.WriteTo(w)
}

func (c *streamConn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

func (c *streamConn) ReadFrom(r io.Reader) (int64, error) {
	return c.w.ReadFrom(r)
}

func (c *streamConn) CloseRead() error {
	if c, ok := c.Conn.(closeReader); ok {
		return c.CloseRead()
	}
	return nil
}

func (c *streamConn) CloseWrite() error {
	if c, ok := c.Conn.(closeWriter); ok {
		return c.CloseWrite()
	}
	return nil
}

// NewConn wraps a stream-oriented net.Conn with AEAD protection.
func NewConn(c net.Conn, aead cipher.AEAD) net.Conn {
	r := &reader{Reader: c, AEAD: aead}
	w := &writer{Writer: c, AEAD: aead}
	return &streamConn{Conn: c, r: r, w: w}
}

// increment little-endian encoded unsigned integer b. Wrap around on overflow.
func increment(b []byte) {
	for i := range b {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}
