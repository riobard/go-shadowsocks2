package shadowstream

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"net"
)

const bufSize = 32 * 1024

type writer struct {
	io.Writer
	Cipher
	cipher.Stream
	buf []byte
}

// NewWriter wraps an io.Writer with stream cipher encryption.
func NewWriter(w io.Writer, s Cipher) io.Writer {
	return &writer{
		Writer: w,
		Cipher: s,
	}
}

func (w *writer) ReadFrom(r io.Reader) (n int64, err error) {
	if w.Stream == nil {
		w.buf = make([]byte, bufSize)
		iv := w.buf[:w.IVSize()]
		if _, err = io.ReadFull(rand.Reader, iv); err != nil {
			return
		}
		if _, err = w.Writer.Write(iv); err != nil {
			return
		}

		w.Stream = w.Encrypter(iv)
	}

	for {
		buf := w.buf
		nr, er := r.Read(buf)
		if nr > 0 {
			n += int64(nr)
			buf = buf[:nr]
			w.XORKeyStream(buf, buf)
			_, ew := w.Writer.Write(buf)
			if ew != nil {
				err = ew
				return
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.ReaderFrom contract
				err = er
			}
			return
		}
	}
}

func (w *writer) Write(b []byte) (int, error) {
	n, err := w.ReadFrom(bytes.NewBuffer(b))
	return int(n), err
}

type reader struct {
	io.Reader
	Cipher
	cipher.Stream
	buf []byte
}

// NewReader wraps an io.Reader with stream cipher decryption.
func NewReader(r io.Reader, s Cipher) io.Reader {
	return &reader{Reader: r, Cipher: s}
}

func (r *reader) Read(b []byte) (int, error) {
	if r.Stream == nil {
		r.buf = make([]byte, bufSize)
		iv := make([]byte, r.IVSize())
		if _, err := io.ReadFull(r.Reader, iv); err != nil {
			return 0, err
		}

		r.Stream = r.Decrypter(iv)
	}

	n, err := r.Reader.Read(b)
	if err != nil {
		return 0, err
	}
	b = b[:n]
	r.XORKeyStream(b, b)
	return n, nil
}

func (r *reader) WriteTo(w io.Writer) (n int64, err error) {
	for {
		buf := r.buf
		nr, er := r.Read(buf)
		if nr > 0 {
			nw, ew := w.Write(buf[:nr])
			n += int64(nw)

			if ew != nil {
				err = ew
				return
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.Copy contract (using src.WriteTo shortcut)
				err = er
			}
			return
		}
	}
}

type conn struct {
	net.Conn
	r *reader
	w *writer
}

// NewConn wraps a stream-oriented net.Conn with stream cipher encryption/decryption.
func NewConn(c net.Conn, ciph Cipher) net.Conn {
	r := &reader{Reader: c, Cipher: ciph}
	w := &writer{Writer: c, Cipher: ciph}
	return &conn{Conn: c, r: r, w: w}
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *conn) WriteTo(w io.Writer) (int64, error) {
	return c.r.WriteTo(w)
}

func (c *conn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	return c.w.ReadFrom(r)
}

type closeWriter interface {
	CloseWrite() error
}

type closeReader interface {
	CloseRead() error
}

func (c *conn) CloseRead() error {
	if c, ok := c.Conn.(closeReader); ok {
		return c.CloseRead()
	}
	return nil
}

func (c *conn) CloseWrite() error {
	if c, ok := c.Conn.(closeWriter); ok {
		return c.CloseWrite()
	}
	return nil
}
