package main

/*
 * context.go
 * HTTP query context
 * By J. Stuart McMurray
 * Created 20230713
 * Last Modified 20230713
 */

import "context"

// ctxKey is used to store values in a context.
type ctxKey int

const (
	cancelFuncKey ctxKey = iota
)

// NewContext returns a new context, ready for use.  If parent is not nil, it
// will be used as the parent context.
func NewContext(parent context.Context) context.Context {
	/* If we don't have a better parent, use the background. */
	if nil == parent {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	return context.WithValue(ctx, cancelFuncKey, cancel)
}

// Cancel cancels a context generated from NewContext with the given error.
func Cancel(ctx context.Context) {
	/* This is weird go. */
	ctx.Value(cancelFuncKey).(context.CancelFunc)()
	/* Contexts are weird. */
}
