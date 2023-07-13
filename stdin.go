package main

/*
 * stdin.go
 * Read from stdin
 * By J. Stuart McMurray
 * Created 20230713
 * Last Modified 20230713
 */

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/sync/semaphore"
)

// bufLen is the length of data to try to read from Stdin in one go.
const bufLen = 1024

var (
	/* Stdin contains bytes read from stdin.  Received slices should be
	returned with ReturnStdinSlice.  Stdin will be closed when reading from
	stdin errors. */
	Stdin = make(chan []byte)

	/* StdinErr will contain the error which caused Stdin to be closed.  It
	can also be used to check if Stdin is dead without a read from
	Stdin. */
	StdinErr atomic.Pointer[error] /* really an error */

	/* StdinSem makes sure only one goroutine reads from stdin at once. */
	StdinSem = semaphore.NewWeighted(1)
)

var (
	/* stdinPool is our stdin read buffer pool. */
	stdinPool = sync.Pool{New: func() any {
		b := make([]byte, bufLen)
		return &b
	}}
)

// ReadStdin starts reading from stdin and making chunks available on Stdin.
func ReadStdin() {
	defer close(Stdin)
	for {
		/* Get and reset a stdin buffer. */
		buf := *(stdinPool.Get().(*[]byte))
		buf = buf[:cap(buf)]

		/* If we can read anything, make it available. */
		nr, err := os.Stdin.Read(buf)
		if 0 != nr {
			Stdin <- buf[:nr]
		}

		/* Kooky old stdlib quirk. */
		if nil != err {
			if !errors.Is(err, io.EOF) {
				log.Printf("Error reading stdin: %s", err)
			}
			StdinErr.Store(&err)
			return
		}
	}
}

// returnStdinSlice collects the Pfand from a used stdin slice.
func returnStdinSlice(s []byte) { stdinPool.Put(&s) }

// copyStdin copies from stdin to w, flushing as it goes.  Headers will be
// set on w to indicate this is a stream.
func copyStdin(ctx context.Context, w http.ResponseWriter) (int, error) {
	var tot int

	/* Set headers to prep for stream output. */
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	/* Prep for flushing. */
	rc := http.NewResponseController(w)

	/* Watch for SIGHUP. */
	hupch := make(chan os.Signal, 2)
	signal.Notify(hupch, syscall.SIGHUP)
	defer signal.Stop(hupch)

	/* Get chunks of stdin and send to the client. */
	for {
		select {
		case <-ctx.Done(): /* Context cancelled. */
			return tot, context.Cause(ctx)
		case b, ok := <-Stdin: /* Read from stdin. */
			if !ok { /* Stdin died. */
				return tot, fmt.Errorf(
					"reading stdin: %w",
					*StdinErr.Load(),
				)
			}
			/* Send it off. */
			n, err := w.Write(b)
			returnStdinSlice(b)
			tot += n
			if nil != err {
				return tot, fmt.Errorf(
					"sending response: %w",
					err,
				)
			}
			/* Try to make sure it gets there. */
			if err := rc.Flush(); nil != err && !errors.Is(
				err,
				http.ErrNotSupported,
			) {
				return tot, fmt.Errorf(
					"flushing response: %w",
					err,
				)
			}
		case <-hupch: /* Got SIGHUP. */
			return tot, ErrSIGHUP
		}
	}
}
