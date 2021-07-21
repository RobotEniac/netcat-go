package util

import (
	"errors"
	"io"
)

type StringWriter struct {
	c chan string
}

func MakeStringWriter() *StringWriter {
	sw := &StringWriter{
		c: make(chan string),
	}
	return sw
}

func (r *StringWriter) Read(p []byte) (n int, err error) {
	if r.c == nil {
		return 0, errors.New("channel is nil")
	}
	msg, ok := <- r.c
	if !ok {
		return 0, io.EOF
	}
	n = copy(p, []byte(msg))
	return n, nil
}

func (r *StringWriter) Write(p []byte) (n int, err error) {
	r.c <- string(p)
	return len(p), nil
}

func (r *StringWriter) WriteString(s string) {
	r.c <- s
}

func (r *StringWriter) Close() error {
	close(r.c)
	return nil
}
