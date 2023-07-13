package main

/*
 * handle.go
 * Handle HTTP queries
 * By J. Stuart McMurray
 * Created 20230713
 * Last Modified 20230713
 */

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	/* ErrSIGHUP indicates we got a SIGHUP. */
	ErrSIGHUP = errors.New("got SIGHUP")
)

// HandleIn handles a connection for input to the remote shell.
func HandleIn(w http.ResponseWriter, r *http.Request) {
	/* Actually do the transfer.  And lots of error-handling. */
	log.Printf("[%s] New stdin connection", r.RemoteAddr)
	handleIn(NewContext(r.Context()), w, r)
}

// handleIn shuffles bytes from Stdin to w.
func handleIn(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	defer Cancel(ctx)

	/* Don't double-read. */
	var waited bool
	if !StdinSem.TryAcquire(1) {
		log.Printf("[%s] Waiting to grab stdin", r.RemoteAddr)
		waited = true
		if err := StdinSem.Acquire(ctx, 1); nil != err {
			log.Printf(
				"[%s] Error waiting to grab stdin: %s",
				r.RemoteAddr,
				err,
			)
			Cancel(ctx)
		}
	}
	defer StdinSem.Release(1)

	/* Don't bother if stdin is closed. */
	if errp := StdinErr.Load(); nil != errp {
		log.Printf(
			"[%s] Refusing stdin connection (stdin error: %s)",
			r.RemoteAddr,
			*errp,
		)
		return
	}

	/* If we waited this long, tell the user it's our turn. */
	if waited {
		log.Printf("[%s] Grabbed stdin", r.RemoteAddr)
	}

	/* Actually do the copy. */
	n, err := copyStdin(ctx, w)
	switch {
	case errors.Is(err, io.EOF): /* "Normal" end */
		log.Printf(
			"[%s] EOF on stdin after %d bytes",
			r.RemoteAddr,
			n,
		)
	case errors.Is(err, ErrSIGHUP):
		log.Printf(
			"[%s] Closing stdin on SIGHUP after %d bytes",
			r.RemoteAddr,
			n,
		)
	case errors.Is(err, context.Canceled):
		log.Printf(
			"[%s] End of stdin stream after %d bytes",
			r.RemoteAddr,
			n,
		)
	case nil == err: /* One day I'll write code this clean. */
		log.Printf(
			"[%s] Unpossible error-free stdin copy after %d bytes",
			r.RemoteAddr,
			n,
		)
	default: /* Some other error. */
		log.Printf(
			"[%s] Error on stdin stream after %d bytes: %s",
			r.RemoteAddr,
			n,
			err,
		)
	}
}

// HandleOut sends r's body to Stdout.
func HandleOut(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] New stdout connection", r.RemoteAddr)
	handleOut(os.Stdout, "stdout", r)
}

// HandleErr send's r's body to Stderr.
func HandleErr(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] New stderr connection", r.RemoteAddr)
	handleOut(os.Stderr, "stderr", r)
}

// HandleBidir handles a bidirectional HTTP/2 stream.
func HandleBidir(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] New bidirectional connection", r.RemoteAddr)

	/* Parallelism is hard. */
	ctx := NewContext(r.Context())
	defer Cancel(ctx)

	/* Do the transfering. */
	go func() {
		defer Cancel(ctx)
		handleOut(os.Stdout, "stdout", r)
	}()
	handleIn(ctx, w, r)
}

// handleOut handles writing output to w, named n.
func handleOut(w io.Writer, n string, r *http.Request) {
	var (
		nw  int64
		err error
	)
	defer func() {
		if nil != err && !errors.Is(err, io.EOF) {
			log.Printf(
				"[%s] Error on %s stream after %d bytes: %s",
				r.RemoteAddr,
				n,
				nw,
				err,
			)
		} else {
			log.Printf(
				"[%s] End of %s stream after %d bytes",
				r.RemoteAddr,
				n,
				nw,
			)
		}
	}()

	/* Actually do the transfer. */
	nw, err = io.Copy(w, r.Body)
}
